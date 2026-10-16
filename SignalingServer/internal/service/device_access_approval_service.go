package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// DeviceAccessApprovalDefaultTTL is how long the owner has to decide before a
// pending request expires. Short on purpose: the controller is sitting in
// front of a countdown.
const DeviceAccessApprovalDefaultTTL = 2 * time.Minute

// Access-request queue limits.
//
// A correct access code is not a licence to fill the owner's decision queue.
// Without these guards one source can create unlimited pending requests and
// every one of them pushes a notification, so the "second factor" becomes a
// way to page the owner at will.
const (
	// MaxPendingApprovalsPerDevice bounds how many unanswered requests one
	// device accumulates, whatever the sources. The trade-off is explicit: a
	// distributed source can hold the queue full for up to one TTL, which is
	// why the cap is paired with a short TTL and not set to 1.
	MaxPendingApprovalsPerDevice = 5

	// approvalDenialStreakScanLimit bounds how far back the escalating denial
	// cooldown counts. A source that has been refused this many times in a row
	// is already pinned to the ceiling by any multiplier worth configuring, so
	// reading further would only cost queries.
	approvalDenialStreakScanLimit = 64

	// denialCooldownSaturation is the largest representable Duration. The
	// escalation multiplies, so it has to stop somewhere: a streak long enough
	// to overflow is a streak that should be capped by configuration anyway.
	denialCooldownSaturation = time.Duration(1<<63 - 1)
)

// Reasons a request was refused instead of queued. Stable strings so the API
// can echo them and a client can branch on them.
const (
	// ApprovalRefusedAlreadyPending means this same client already has a
	// request waiting for this device.
	ApprovalRefusedAlreadyPending = "already_pending"
	// ApprovalRefusedQueueFull means the device's decision queue is full.
	ApprovalRefusedQueueFull = "queue_full"
	// ApprovalRefusedDeniedRecently means this source's last request for this
	// device was denied by the owner inside the cooldown window. Without it a
	// denial clears the way for the next request, so a client holding a correct
	// code can page the owner again the moment they say no.
	ApprovalRefusedDeniedRecently = "denied_recently"
)

// PendingApprovalError reports that no approval was created because one is
// already waiting. Callers map it onto 429 with a retry hint rather than a
// 5xx: the request was well-formed and the code was right, the client simply
// has to wait.
type PendingApprovalError struct {
	Reason string
	// RetryAfter is how long until the blocking request stops mattering.
	RetryAfter time.Duration
}

func (e *PendingApprovalError) Error() string {
	if e.Reason == ApprovalRefusedQueueFull {
		return "too many access requests are already waiting for the owner"
	}
	if e.Reason == ApprovalRefusedDeniedRecently {
		return "access was recently denied for this client; wait before asking again"
	}
	return "an access request from this client is already waiting for the owner"
}

// ClaimOutcome is the state a claim attempt observed. It mirrors the approval
// status plus the two transport-level cases (not found / already consumed) so
// the handler can map straight onto HTTP codes.
type ClaimOutcome int

const (
	// ClaimConfirmed means the approval had been granted and this call just
	// consumed it: the caller may mint a signal token.
	ClaimConfirmed ClaimOutcome = iota
	// ClaimPending means the owner has not decided yet.
	ClaimPending
	// ClaimDenied means the owner refused.
	ClaimDenied
	// ClaimExpired means no decision arrived in time.
	ClaimExpired
	// ClaimConsumed means the approval was already exchanged for a token.
	ClaimConsumed
	// ClaimNotFound means the id/secret pair does not match an approval. It is
	// deliberately indistinguishable from a wrong secret so the endpoint is
	// not an id oracle.
	ClaimNotFound
)

// DeviceAccessApprovalService issues and resolves second-factor approvals.
type DeviceAccessApprovalService struct {
	db *gorm.DB
	// denialCooldown, when positive, refuses a new request from a source whose
	// last request for the same device was denied inside the window. 0 disables
	// it, which is the historical behaviour: a denial immediately frees the
	// source to ask again.
	denialCooldown time.Duration
	// denialCooldownMultiplier lengthens the window for a source that has been
	// denied repeatedly: the streak of leading denials is the exponent. 1
	// leaves the window constant, which is the historical behaviour.
	denialCooldownMultiplier int
	// denialCooldownMax, when positive, is the ceiling the escalation cannot
	// pass. 0 means "no ceiling", so only the multiplier bounds the window.
	denialCooldownMax time.Duration
}

func NewDeviceAccessApprovalService(db *gorm.DB) *DeviceAccessApprovalService {
	return &DeviceAccessApprovalService{db: db, denialCooldownMultiplier: 1}
}

// WithDenialCooldown sets how long a denied source must wait before it may ask
// the same device again. Negative values are treated as "disabled".
func (s *DeviceAccessApprovalService) WithDenialCooldown(d time.Duration) *DeviceAccessApprovalService {
	if d < 0 {
		d = 0
	}
	s.denialCooldown = d
	return s
}

// DenialCooldown reports the configured window (0 = disabled).
func (s *DeviceAccessApprovalService) DenialCooldown() time.Duration {
	if s == nil {
		return 0
	}
	return s.denialCooldown
}

// WithDenialCooldownEscalation makes repeated denials cost more: a source that
// has been refused `n` times in a row without a decision in its favour waits
// `denialCooldown * multiplier^(n-1)`, capped at maxSeconds.
//
// It is deliberately additive to WithDenialCooldown rather than a replacement:
// a multiplier of 1 and a ceiling of 0 reproduce the fixed window exactly, so
// an existing deployment that sets neither keeps the behaviour it has.
func (s *DeviceAccessApprovalService) WithDenialCooldownEscalation(multiplier int, maxSeconds int) *DeviceAccessApprovalService {
	if multiplier < 1 {
		multiplier = 1
	}
	ceiling := time.Duration(0)
	if maxSeconds > 0 {
		ceiling = time.Duration(maxSeconds) * time.Second
	}
	s.denialCooldownMultiplier = multiplier
	s.denialCooldownMax = ceiling
	return s
}

// DenialCooldownMultiplier reports the configured escalation factor (1 =
// escalation disabled).
func (s *DeviceAccessApprovalService) DenialCooldownMultiplier() int {
	if s == nil || s.denialCooldownMultiplier < 1 {
		return 1
	}
	return s.denialCooldownMultiplier
}

// DenialCooldownMax reports the configured ceiling (0 = uncapped).
func (s *DeviceAccessApprovalService) DenialCooldownMax() time.Duration {
	if s == nil {
		return 0
	}
	return s.denialCooldownMax
}

// denialWindowFor returns how long a source with this denial streak must wait.
// The ceiling applies whenever it is set, including with a multiplier of 1,
// where it acts as a plain override rather than an escalation cap.
func (s *DeviceAccessApprovalService) denialWindowFor(streak int) time.Duration {
	base := s.DenialCooldown()
	if base <= 0 {
		return 0
	}
	window := base
	if multiplier := s.DenialCooldownMultiplier(); multiplier > 1 {
		for i := 1; i < streak; i++ {
			if ceiling := s.DenialCooldownMax(); ceiling > 0 && window >= ceiling {
				break
			}
			step := time.Duration(multiplier)
			if window > denialCooldownSaturation/step {
				window = denialCooldownSaturation
				break
			}
			window *= step
		}
	}
	if ceiling := s.DenialCooldownMax(); ceiling > 0 && window > ceiling {
		window = ceiling
	}
	return window
}

// CreateDeviceAccessApprovalInput is one pending request.
type CreateDeviceAccessApprovalInput struct {
	DeviceID string
	OwnerID  uint
	ClientID string
	ClientIP string
	Source   string
	GrantID  *uint
	TTL      time.Duration
}

// Create stores a pending approval and returns it together with the one-time
// plaintext secret the caller must hand back on claim.
func (s *DeviceAccessApprovalService) Create(
	ctx context.Context,
	in CreateDeviceAccessApprovalInput,
) (*models.DeviceAccessApproval, string, error) {
	if strings.TrimSpace(in.DeviceID) == "" {
		return nil, "", errors.New("device_id is required")
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = models.ApprovalSourceDeviceCode
	}
	if source != models.ApprovalSourceDeviceCode && source != models.ApprovalSourceGrant {
		return nil, "", fmt.Errorf("invalid approval source %q", source)
	}
	ttl := in.TTL
	if ttl <= 0 {
		ttl = DeviceAccessApprovalDefaultTTL
	}
	secret, err := generateApprovalSecret()
	if err != nil {
		return nil, "", err
	}
	now := time.Now().UTC()
	row := &models.DeviceAccessApproval{
		DeviceID:   in.DeviceID,
		OwnerID:    in.OwnerID,
		ClientID:   in.ClientID,
		ClientIP:   in.ClientIP,
		Source:     source,
		GrantID:    in.GrantID,
		SecretHash: HashApprovalSecret(secret),
		Status:     models.ApprovalPending,
		ExpiresAt:  now.Add(ttl),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.guardPendingRequests(tx, in, now); err != nil {
			return err
		}
		return tx.Create(row).Error
	})
	if err != nil {
		return nil, "", err
	}
	return row, secret, nil
}

// guardPendingRequests refuses a new request while one is already waiting.
//
// Two guards, for two different abuses. The first is per source: the same
// client asking twice does not create two entries in the owner's queue, which
// is what "连续发起多笔审批" was. The second is per device: whatever the
// sources, the queue has a ceiling, because every entry is also a notification
// and "unbounded queue" means "unbounded ability to page the owner".
//
// It runs inside the caller's transaction and takes a row lock on the device
// first. A bare check-then-insert is exactly the shape that races: two
// simultaneous requests from the same source would both see an empty queue and
// both insert. The lock serialises them per device, and it is short - two
// indexed reads and one insert.
func (s *DeviceAccessApprovalService) guardPendingRequests(
	tx *gorm.DB,
	in CreateDeviceAccessApprovalInput,
	now time.Time,
) error {
	var device struct {
		ID uint `gorm:"column:id"`
	}
	if err := tx.Raw("SELECT id FROM devices WHERE device_id = ? FOR UPDATE", in.DeviceID).
		Scan(&device).Error; err != nil {
		return err
	}

	// A stale `pending` row must not block a legitimate request, and the owner
	// may not have opened the approvals list since it lapsed.
	if err := tx.Model(&models.DeviceAccessApproval{}).
		Where("device_id = ? AND status = ? AND expires_at <= ?", in.DeviceID, models.ApprovalPending, now).
		Updates(map[string]interface{}{
			"status":     models.ApprovalExpired,
			"decided_at": now,
			"reason":     "expired without a decision",
			"updated_at": now,
		}).Error; err != nil {
		return err
	}

	if ip := strings.TrimSpace(in.ClientIP); ip != "" {
		var existing models.DeviceAccessApproval
		err := tx.Where("device_id = ? AND client_ip = ? AND status = ?", in.DeviceID, ip, models.ApprovalPending).
			Order("id ASC").
			First(&existing).Error
		switch {
		case err == nil:
			return &PendingApprovalError{
				Reason:     ApprovalRefusedAlreadyPending,
				RetryAfter: time.Until(existing.ExpiresAt),
			}
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}
	}

	// A denial has to mean something. Without this, "no" merely clears the
	// pending slot and the same source can ask again on the next tick, turning
	// a correct access code into a way to keep paging the owner. The window is
	// per (device, source), so one stranger's denial cannot lock out anybody
	// else, and the retry hint is honest about when the block lifts.
	//
	// The window grows with the streak of consecutive denials ("no" after "no"
	// costs more than the first "no") and never shrinks below the configured
	// base. An approval, or a request that simply expired because the owner
	// never answered, ends the streak: only an explicit refusal escalates.
	if ip := strings.TrimSpace(in.ClientIP); ip != "" && s.denialCooldown > 0 {
		streak, lastDeniedAt, err := s.denialStreak(tx, in.DeviceID, ip)
		if err != nil {
			return err
		}
		if streak > 0 && lastDeniedAt != nil {
			if retry := time.Until(lastDeniedAt.Add(s.denialWindowFor(streak))); retry > 0 {
				return &PendingApprovalError{
					Reason:     ApprovalRefusedDeniedRecently,
					RetryAfter: retry,
				}
			}
		}
	}

	var pending int64
	if err := tx.Model(&models.DeviceAccessApproval{}).
		Where("device_id = ? AND status = ?", in.DeviceID, models.ApprovalPending).
		Count(&pending).Error; err != nil {
		return err
	}
	if pending >= MaxPendingApprovalsPerDevice {
		var next models.DeviceAccessApproval
		_ = tx.Where("device_id = ? AND status = ?", in.DeviceID, models.ApprovalPending).
			Order("expires_at ASC").
			First(&next).Error
		return &PendingApprovalError{
			Reason:     ApprovalRefusedQueueFull,
			RetryAfter: time.Until(next.ExpiresAt),
		}
	}
	return nil
}

// denialStreak counts how many times this source has been refused in a row
// without a decision in its favour, and reports when the most recent refusal
// happened.
//
// The streak is read from the decided rows for this (device, source) pair in
// reverse chronological order and stops at the first decision that was not a
// denial. That makes the counter self-healing: an approval, or a request that
// merely expired because nobody answered, resets it to zero, so a source that
// once misbehaved is not punished months later. Only a refusal escalates.
func (s *DeviceAccessApprovalService) denialStreak(
	tx *gorm.DB,
	deviceID string,
	ip string,
) (int, *time.Time, error) {
	var rows []models.DeviceAccessApproval
	if err := tx.
		Where("device_id = ? AND client_ip = ? AND decided_at IS NOT NULL", deviceID, ip).
		Order("decided_at DESC, id DESC").
		Limit(approvalDenialStreakScanLimit).
		Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	streak := 0
	for _, row := range rows {
		if row.Status != models.ApprovalDenied {
			break
		}
		streak++
	}
	if streak == 0 {
		return 0, nil, nil
	}
	// Rows are ordered newest first and the query excludes undecided rows, so
	// this is the timestamp of the refusal that is currently holding the door.
	return streak, rows[0].DecidedAt, nil
}

// Get returns one approval by id, or (nil, nil) when it does not exist.
func (s *DeviceAccessApprovalService) Get(ctx context.Context, approvalID uint) (*models.DeviceAccessApproval, error) {
	var row models.DeviceAccessApproval
	err := s.db.WithContext(ctx).Where("id = ?", approvalID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// List returns a device's approvals newest-first. A non-empty statuses filter
// narrows to those statuses; overdue pending rows are expired first so a
// "pending" list never shows something that already timed out. status filter
// `expired` therefore also matches rows that were pending a moment ago.
func (s *DeviceAccessApprovalService) List(
	ctx context.Context,
	deviceID string,
	statuses []string,
	limit int,
) ([]models.DeviceAccessApproval, error) {
	if err := s.ExpireOverdue(ctx, deviceID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultApprovalListLimit
	}
	if limit > maxApprovalListLimit {
		limit = maxApprovalListLimit
	}
	q := s.db.WithContext(ctx).Model(&models.DeviceAccessApproval{}).Where("device_id = ?", deviceID)
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	var rows []models.DeviceAccessApproval
	err := q.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// Bounds for the owner-facing summary view.
const (
	// ApprovalSummaryDefaultWindow is how far back the summary looks when the
	// caller does not say.
	ApprovalSummaryDefaultWindow = 24 * time.Hour
	// ApprovalSummaryMaxSources caps how many refused sources come back. The
	// view exists to spot a source hammering the owner, not to export the table.
	ApprovalSummaryMaxSources = 10
	// approvalSummaryDefaultSources is the default cap when none is given.
	approvalSummaryDefaultSources = 5
)

// ApprovalDeniedSource is one refused source, aggregated.
type ApprovalDeniedSource struct {
	ClientIP string `json:"client_ip"`
	Denials  int64  `json:"denials"`
	// LastDeniedAt is when that source was most recently refused.
	LastDeniedAt *time.Time `json:"last_denied_at,omitempty"`
}

// ApprovalSummary answers "what has this device's approval queue been doing",
// which the row list cannot: a list shows individual requests, so the owner has
// to count them by eye to notice that one address has been refused five times
// this afternoon.
type ApprovalSummary struct {
	// Since is the inclusive start of the window the counts cover.
	Since time.Time `json:"since"`
	// Total is every request created inside the window, whatever its outcome.
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
	// DeniedSources is the refused sources inside the window, most refusals
	// first. It is what makes "someone keeps trying" visible.
	DeniedSources []ApprovalDeniedSource `json:"denied_sources"`
	// DenialCooldownSeconds echoes the configured cooldown so the owner can
	// tell "this source is being refused right now" from "it was refused a
	// while ago".
	DenialCooldownSeconds int `json:"denial_cooldown_seconds"`
	// DenialCooldownMultiplier echoes how much each further consecutive
	// refusal lengthens the window (1 = the window never grows).
	DenialCooldownMultiplier int `json:"denial_cooldown_multiplier"`
	// DenialCooldownMaxSeconds echoes the ceiling the escalation cannot pass
	// (0 = uncapped).
	DenialCooldownMaxSeconds int `json:"denial_cooldown_max_seconds"`
}

// Summary aggregates this device's approvals inside a window. The window is
// keyed on `created_at` for every count, so "requests that arrived in this
// window" is the one definition: a denial of a request from before the window
// is not counted, even if the decision happened after it started.
func (s *DeviceAccessApprovalService) Summary(
	ctx context.Context,
	deviceID string,
	since time.Time,
	limit int,
) (ApprovalSummary, error) {
	if err := s.ExpireOverdue(ctx, deviceID); err != nil {
		return ApprovalSummary{}, err
	}
	if limit <= 0 {
		limit = approvalSummaryDefaultSources
	}
	if limit > ApprovalSummaryMaxSources {
		limit = ApprovalSummaryMaxSources
	}
	out := ApprovalSummary{
		Since: since,
		// Every state is present even at zero, so a client never has to guess
		// whether a missing key means "none" or "unknown".
		ByStatus: map[string]int64{
			models.ApprovalPending:  0,
			models.ApprovalApproved: 0,
			models.ApprovalDenied:   0,
			models.ApprovalExpired:  0,
			models.ApprovalConsumed: 0,
		},
		DeniedSources:            []ApprovalDeniedSource{},
		DenialCooldownSeconds:    int(s.denialCooldown.Seconds()),
		DenialCooldownMultiplier: s.DenialCooldownMultiplier(),
		DenialCooldownMaxSeconds: int(s.DenialCooldownMax().Seconds()),
	}

	type statusRow struct {
		Status string
		Total  int64
	}
	var statusRows []statusRow
	if err := s.db.WithContext(ctx).
		Model(&models.DeviceAccessApproval{}).
		Select("status, count(*) AS total").
		Where("device_id = ? AND created_at >= ?", deviceID, since).
		Group("status").
		Scan(&statusRows).Error; err != nil {
		return out, err
	}
	for _, r := range statusRows {
		out.ByStatus[r.Status] = r.Total
		out.Total += r.Total
	}

	type sourceRow struct {
		ClientIP     string
		Denials      int64
		LastDeniedAt *time.Time
	}
	var sourceRows []sourceRow
	if err := s.db.WithContext(ctx).
		Model(&models.DeviceAccessApproval{}).
		Select("client_ip, count(*) AS denials, max(decided_at) AS last_denied_at").
		Where("device_id = ? AND status = ? AND created_at >= ?", deviceID, models.ApprovalDenied, since).
		Group("client_ip").
		Order("denials DESC, last_denied_at DESC").
		Limit(limit).
		Scan(&sourceRows).Error; err != nil {
		return out, err
	}
	for _, r := range sourceRows {
		out.DeniedSources = append(out.DeniedSources, ApprovalDeniedSource{
			ClientIP:     r.ClientIP,
			Denials:      r.Denials,
			LastDeniedAt: r.LastDeniedAt,
		})
	}
	return out, nil
}

// ExpireOverdue flips this device's unanswered, past-deadline rows to
// `expired`. It is safe to call repeatedly and is the only way a pending row
// changes without an owner action.
func (s *DeviceAccessApprovalService) ExpireOverdue(ctx context.Context, deviceID string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).
		Model(&models.DeviceAccessApproval{}).
		Where("device_id = ? AND status = ? AND expires_at <= ?", deviceID, models.ApprovalPending, now).
		Updates(map[string]interface{}{
			"status":     models.ApprovalExpired,
			"decided_at": now,
			"reason":     "expired without a decision",
			"updated_at": now,
		}).Error
}

// Decide records the owner's verdict. It returns the updated row and whether
// this call changed anything: deciding an already-decided row is a no-op, so a
// double-click cannot flip an approval.
func (s *DeviceAccessApprovalService) Decide(
	ctx context.Context,
	deviceID string,
	approvalID uint,
	approve bool,
	decidedBy uint,
	reason string,
) (*models.DeviceAccessApproval, bool, error) {
	now := time.Now().UTC()
	status := models.ApprovalDenied
	if approve {
		status = models.ApprovalApproved
	}
	if len(reason) > 255 {
		reason = reason[:255]
	}
	result := s.db.WithContext(ctx).
		Model(&models.DeviceAccessApproval{}).
		Where("id = ? AND device_id = ? AND status = ? AND expires_at > ?",
			approvalID, deviceID, models.ApprovalPending, now).
		Updates(map[string]interface{}{
			"status":     status,
			"decided_by": decidedBy,
			"decided_at": now,
			"reason":     reason,
			"updated_at": now,
		})
	if result.Error != nil {
		return nil, false, result.Error
	}
	row, err := s.Get(ctx, approvalID)
	if err != nil {
		return nil, false, err
	}
	if row == nil || row.DeviceID != deviceID {
		return nil, false, nil
	}
	return row, result.RowsAffected == 1, nil
}

// Peek verifies the one-time secret and reports the current state without
// consuming anything. The handler uses it to check that the host is still
// online before it burns the approval.
//
// A missing id and a wrong secret both return ClaimNotFound on purpose: the
// endpoint must not become an id oracle.
func (s *DeviceAccessApprovalService) Peek(
	ctx context.Context,
	deviceID string,
	approvalID uint,
	secret string,
) (*models.DeviceAccessApproval, ClaimOutcome, error) {
	row, err := s.Get(ctx, approvalID)
	if err != nil {
		return nil, ClaimNotFound, err
	}
	if row == nil || row.DeviceID != deviceID || !ApprovalSecretMatches(row.SecretHash, secret) {
		return nil, ClaimNotFound, nil
	}
	now := time.Now().UTC()
	switch row.EffectiveStatusAt(now) {
	case models.ApprovalPending:
		return row, ClaimPending, nil
	case models.ApprovalDenied:
		return row, ClaimDenied, nil
	case models.ApprovalExpired:
		// Best-effort persistence; the caller already has the answer.
		_ = s.db.WithContext(ctx).
			Model(&models.DeviceAccessApproval{}).
			Where("id = ? AND status = ?", row.ID, models.ApprovalPending).
			Updates(map[string]interface{}{
				"status": models.ApprovalExpired, "decided_at": now,
				"updated_at": now, "reason": "expired without a decision",
			}).Error
		return row, ClaimExpired, nil
	case models.ApprovalConsumed:
		return row, ClaimConsumed, nil
	case models.ApprovalApproved:
		return row, ClaimConfirmed, nil
	default:
		return row, ClaimDenied, nil
	}
}

// Claim verifies the one-time secret and, when the approval was granted,
// performs the `approved -> consumed` transition in the UPDATE itself, so two
// controllers racing the same approval cannot both mint a signal token:
// exactly one sees ClaimConfirmed, the other sees ClaimConsumed.
func (s *DeviceAccessApprovalService) Claim(
	ctx context.Context,
	deviceID string,
	approvalID uint,
	secret string,
) (*models.DeviceAccessApproval, ClaimOutcome, error) {
	row, outcome, err := s.Peek(ctx, deviceID, approvalID, secret)
	if err != nil || outcome != ClaimConfirmed {
		return row, outcome, err
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).
		Model(&models.DeviceAccessApproval{}).
		Where("id = ? AND status = ? AND consumed_at IS NULL", row.ID, models.ApprovalApproved).
		Updates(map[string]interface{}{
			"status":      models.ApprovalConsumed,
			"consumed_at": now,
			"updated_at":  now,
		})
	if result.Error != nil {
		return nil, ClaimNotFound, result.Error
	}
	if result.RowsAffected != 1 {
		return row, ClaimConsumed, nil
	}
	row.Status = models.ApprovalConsumed
	row.ConsumedAt = &now
	return row, ClaimConfirmed, nil
}

const (
	defaultApprovalListLimit = 50
	maxApprovalListLimit     = 200
)

// generateApprovalSecret returns 32 bytes of hex (64 chars). It is the bearer
// proof a controller presents to poll and claim its own approval.
func generateApprovalSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate approval secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// HashApprovalSecret is the stored form of an approval secret.
func HashApprovalSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// ApprovalSecretMatches compares the presented secret against a stored hash in
// constant time.
func ApprovalSecretMatches(storedHash, presented string) bool {
	if storedHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(HashApprovalSecret(presented))) == 1
}
