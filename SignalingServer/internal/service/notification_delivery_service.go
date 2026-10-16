package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// Delivery states for a notification outbox row.
const (
	DeliveryStatusPending    = "pending"
	DeliveryStatusDelivering = "delivering"
	DeliveryStatusDelivered  = "delivered"
	DeliveryStatusFailed     = "failed"
)

// Header names the outbox puts on every POST so a receiver can deduplicate.
// Delivery is at-least-once: a retry, a bulk replay, or a delivery whose
// response was lost all re-send a body the receiver may already have seen.
// X-Webhook-Delivery-Id is stable for one outbox row across every attempt,
// and X-Webhook-Event-Id is stable for the event itself even if the same
// event were ever enqueued to a channel twice.
const (
	HeaderWebhookDeliveryID = "X-Webhook-Delivery-Id"
	HeaderWebhookEventID    = "X-Webhook-Event-Id"
)

// ErrDeliveryNotRetryable means the caller asked to replay a delivery that is
// already delivered (or not parked as failed), so there is nothing to re-arm.
var ErrDeliveryNotRetryable = errors.New("delivery is not retryable")

// ErrDeliveryNotConfirmable means the caller tried to acknowledge a delivery
// that has not finished yet. A row that is still pending or in flight has not
// reached the endpoint, so "somebody saw this" would be a claim nothing
// supports; the terminal states are the only ones where the question makes
// sense.
var ErrDeliveryNotConfirmable = errors.New("only delivered or failed deliveries can be acknowledged")

const (
	// notificationMaxAttempts is how many failed posts a row gets before it
	// is parked as `failed` (still visible through the API).
	notificationMaxAttempts = 8
	// notificationBackoffCap bounds the exponential backoff.
	notificationBackoffCap = 5 * time.Minute
	// notificationClaimBatch is how many due rows one worker pass takes.
	notificationClaimBatch = 20
	// notificationPerChannelBatch caps how many of those rows may come from a
	// single channel. The worker delivers a batch sequentially, so without this
	// cap one endpoint's backlog fills every batch and every other channel's
	// notifications wait behind it (head-of-line blocking).
	notificationPerChannelBatch = 5
	// notificationWorkerInterval is the queue poll interval.
	notificationWorkerInterval = 2 * time.Second
	// notificationPurgeBatch bounds how many rows one archive-then-delete pass
	// reads into memory.
	notificationPurgeBatch = 1000
)

// NotificationDeliveryService is the durable half of user notifications: it
// stores one outbox row per (event, channel) and retries until the endpoint
// answers 2xx. Delivery is at-least-once — a receiver that errors after
// processing may see the event twice, so consumers should treat
// `event` + `timestamp` + `data.log_id` as an idempotency key.
type NotificationDeliveryService struct {
	db *gorm.DB
	// archiveDir, when set, makes retention append expired rows to a JSONL file
	// before deleting them. Empty means the old behaviour: hard delete.
	archiveDir string
	// archiveColdDir, when set, moves expired archive files there instead of
	// deleting them. Empty keeps the old delete-on-rotation behavior.
	archiveColdDir string
	// aggregateWindow, when positive, holds a new delivery for that long and
	// folds later occurrences of the same (channel, event type, device) into
	// it. 0 keeps the historical one-row-per-event behaviour.
	aggregateWindow time.Duration
	// quiet, when set, answers "is this user asking not to be paged right now".
	// Nil means never quiet, which is what every deployment had before the
	// setting existed.
	quiet *NotificationQuietHoursService
}

func NewNotificationDeliveryService(db *gorm.DB) *NotificationDeliveryService {
	return &NotificationDeliveryService{db: db}
}

// WithQuietHours wires the per-user quiet-hours setting. It is optional on
// purpose: the delivery service is constructed in tests and in tools that have
// no reason to know about the setting, and a nil quiet service means "deliver
// as usual".
func (s *NotificationDeliveryService) WithQuietHours(q *NotificationQuietHoursService) *NotificationDeliveryService {
	s.quiet = q
	return s
}

// WithArchiveDir enables archive-before-delete for retention: expired terminal
// rows are appended to a daily JSONL file under dir, and only deleted once that
// write has been flushed to disk. Returns the receiver so it can be chained.
func (s *NotificationDeliveryService) WithArchiveDir(dir string) *NotificationDeliveryService {
	s.archiveDir = strings.TrimSpace(dir)
	return s
}

// WithAggregateWindow enables burst collapsing for noise reduction. When set,
// a delivery is held for that long before its first attempt, and another event
// with the same (channel, event type, device) that arrives during the hold
// increments the row's occurrence count instead of creating a second row. The
// trade-off is a deliberate delay of up to the window before anything is sent;
// 0 (the default) keeps immediate delivery and one row per event.
func (s *NotificationDeliveryService) WithAggregateWindow(d time.Duration) *NotificationDeliveryService {
	if d < 0 {
		d = 0
	}
	s.aggregateWindow = d
	return s
}

// AggregateWindow reports the configured collapse window (0 = disabled).
func (s *NotificationDeliveryService) AggregateWindow() time.Duration {
	if s == nil {
		return 0
	}
	return s.aggregateWindow
}

// ArchiveDir reports the configured archive directory ("" = plain delete).
func (s *NotificationDeliveryService) ArchiveDir() string {
	return s.archiveDir
}

// WithArchiveColdDir makes archive rotation move expired files into dir rather
// than deleting them. Returns the receiver so it can be chained.
func (s *NotificationDeliveryService) WithArchiveColdDir(dir string) *NotificationDeliveryService {
	s.archiveColdDir = strings.TrimSpace(dir)
	return s
}

// ArchiveColdDir reports the configured cold archive directory ("" = delete).
func (s *NotificationDeliveryService) ArchiveColdDir() string {
	if s == nil {
		return ""
	}
	return s.archiveColdDir
}

// CountByStatus returns one count per delivery status, including zeroes, so an
// operator can see the whole outbox at a glance.
func (s *NotificationDeliveryService) CountByStatus(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Status string
		Total  int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Model(&models.NotificationDelivery{}).
		Select("status, count(*) AS total").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[string]int64{
		DeliveryStatusPending:    0,
		DeliveryStatusDelivering: 0,
		DeliveryStatusDelivered:  0,
		DeliveryStatusFailed:     0,
	}
	for _, r := range rows {
		counts[r.Status] = r.Total
	}
	return counts, nil
}

// Enqueue stores one delivery attempt for a channel. Called from the event
// subscriber, so it must stay cheap: no HTTP here.
func (s *NotificationDeliveryService) Enqueue(ctx context.Context, ch models.UserNotificationChannel, eventType, deviceID string, payload interface{}) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if s.aggregateWindow > 0 {
		merged, mergeErr := s.mergeIntoHeldDelivery(ctx, ch.ID, eventType, deviceID, now)
		if mergeErr != nil {
			// Aggregation is best-effort. Falling back to a new row only means
			// the receiver hears about the event more often than it had to,
			// never that it is lost.
			log.Printf("[notification] aggregate merge for channel %d failed: %v", ch.ID, mergeErr)
		} else if merged {
			return nil
		}
	}
	row := &models.NotificationDelivery{
		ChannelID:     ch.ID,
		UserID:        ch.UserID,
		EventType:     eventType,
		DeviceID:      deviceID,
		Payload:       string(encoded),
		Status:        DeliveryStatusPending,
		NextAttemptAt: now,
	}
	if s.aggregateWindow > 0 {
		// Hold the row so a burst has time to collapse into it. The hold is
		// fixed rather than extended on every merge, so the worst-case delay a
		// receiver sees is the window, not the length of the burst.
		row.NextAttemptAt = now.Add(s.aggregateWindow)
	}
	return s.db.WithContext(ctx).Create(row).Error
}

// mergeIntoHeldDelivery folds one more occurrence into the newest still-held
// row for the same (channel, event type, device) and reports whether it did.
//
// "Held" means the row is still `pending` *and* its next_attempt_at is in the
// future: once the hold lapses the worker is free to claim it, and mutating a
// row that is about to be sent (or is already `delivering`) would either lose
// the increment or send a body that no longer matches its count. Because the
// subquery and the update are one statement, two concurrent enqueues both see
// the same snapshot and exactly one row is incremented per statement.
func (s *NotificationDeliveryService) mergeIntoHeldDelivery(
	ctx context.Context,
	channelID uint,
	eventType, deviceID string,
	now time.Time,
) (bool, error) {
	res := s.db.WithContext(ctx).Exec(`
		UPDATE notification_deliveries
		SET occurrence_count = occurrence_count + 1, updated_at = ?
		WHERE id = (
			SELECT id FROM notification_deliveries
			WHERE channel_id = ? AND event_type = ? AND device_id = ?
			  AND status = ? AND next_attempt_at > ?
			ORDER BY id DESC
			LIMIT 1
		)`,
		now, channelID, eventType, deviceID, DeliveryStatusPending, now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// Run polls the outbox until ctx is cancelled.
func (s *NotificationDeliveryService) Run(ctx context.Context) {
	ticker := time.NewTicker(notificationWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.DrainOnce(ctx)
		}
	}
}

// PurgeTerminal deletes finished rows (delivered or failed) that have not been
// touched since cutoff. In-flight rows (pending/delivering) are never touched,
// so a long backoff is not mistaken for history.
//
// When an archive directory is configured the rows are appended to a daily
// JSONL file first and only deleted after that write lands on disk. A failed
// archive write therefore aborts the purge: retention must not be the step that
// silently destroys the only copy of a delivery history.
func (s *NotificationDeliveryService) PurgeTerminal(ctx context.Context, cutoff time.Time) (int64, error) {
	if s.archiveDir == "" {
		result := s.db.WithContext(ctx).
			Where("status IN ?", terminalDeliveryStatuses()).
			Where("updated_at < ?", cutoff.UTC()).
			Delete(&models.NotificationDelivery{})
		return result.RowsAffected, result.Error
	}

	var rows []models.NotificationDelivery
	if err := s.db.WithContext(ctx).
		Where("status IN ?", terminalDeliveryStatuses()).
		Where("updated_at < ?", cutoff.UTC()).
		Order("id").
		Limit(notificationPurgeBatch).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := s.archiveRows(rows); err != nil {
		return 0, fmt.Errorf("archive notification deliveries before purge: %w", err)
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	// Re-check the status in the delete so a row that went back to pending
	// between the read and the delete is not removed by the archive path.
	result := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("status IN ?", terminalDeliveryStatuses()).
		Delete(&models.NotificationDelivery{})
	return result.RowsAffected, result.Error
}

// archiveRows appends one JSON object per row to the day's archive file and
// fsyncs it, so the bytes are durable before the caller deletes them.
func (s *NotificationDeliveryService) archiveRows(rows []models.NotificationDelivery) error {
	if err := os.MkdirAll(s.archiveDir, 0o755); err != nil {
		return err
	}
	name := fmt.Sprintf("notification-deliveries-%s.jsonl", time.Now().UTC().Format("2006-01-02"))
	path := filepath.Join(s.archiveDir, name)

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(buf.Bytes()); err != nil {
		return err
	}
	return file.Sync()
}

func terminalDeliveryStatuses() []string {
	return []string{DeliveryStatusDelivered, DeliveryStatusFailed}
}

// SweepArchive deletes daily archive files older than archiveRetentionDays and
// reports how many it removed. 0 means "keep the archives forever"; a
// deployment with no archive directory simply has nothing to sweep.
func (s *NotificationDeliveryService) SweepArchive(archiveRetentionDays int) (int, error) {
	result, err := s.SweepArchiveDetailed(archiveRetentionDays)
	return result.Removed, err
}

// SweepArchiveDetailed is SweepArchive with the moved/deleted breakdown.
func (s *NotificationDeliveryService) SweepArchiveDetailed(archiveRetentionDays int) (ArchiveSweepResult, error) {
	if s == nil {
		return ArchiveSweepResult{}, nil
	}
	return sweepArchiveDirDetailed(s.archiveDir, "notification-deliveries", archiveRetentionDays, time.Now().UTC(), s.archiveColdDir)
}

// RunRetention sweeps expired terminal rows on a ticker until ctx is
// cancelled. Two independent bounds are enforced: `retentionDays` deletes
// terminal rows older than the window, and `archiveRetentionDays` deletes the
// daily archive files that window wrote. Either one alone is enough to make the
// sweep run, so a deployment can bound the archive without re-purging rows.
//
// Both default to 0, which keeps the outbox and its archives forever, matching
// the recording retention setting.
func (s *NotificationDeliveryService) RunRetention(ctx context.Context, interval time.Duration, retentionDays, archiveRetentionDays int) {
	if interval <= 0 || (retentionDays <= 0 && archiveRetentionDays <= 0) {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if retentionDays > 0 {
				cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
				removed, err := s.PurgeTerminal(ctx, cutoff)
				if err != nil {
					log.Printf("[notification] retention sweep failed: %v", err)
				} else if removed > 0 {
					log.Printf("[notification] retention purged %d terminal deliveries older than %s",
						removed, cutoff.Format(time.RFC3339))
				}
			}
			if archiveRetentionDays > 0 {
				removed, err := s.SweepArchive(archiveRetentionDays)
				if err != nil {
					log.Printf("[notification] archive sweep failed: %v", err)
				} else if removed > 0 {
					log.Printf("[notification] archive sweep removed %d file(s) older than %d day(s)",
						removed, archiveRetentionDays)
				}
			}
		}
	}
}

// DrainOnce claims and delivers every row that is due right now. Exported so
// tests and maintenance commands can drive the queue directly.
func (s *NotificationDeliveryService) DrainOnce(ctx context.Context) {
	claimed, err := s.claimDue(ctx, notificationClaimBatch)
	if err != nil {
		log.Printf("[notification] claim failed: %v", err)
		return
	}
	for _, row := range claimed {
		if until, hold := s.quietUntil(ctx, row); hold {
			s.holdDelivery(ctx, row, until)
			continue
		}
		s.deliver(ctx, row)
	}
}

// quietUntil asks the quiet-hours setting whether this row's owner is asking
// not to be paged right now.
//
// A lookup failure fails *open*: the alert is delivered and the error is
// logged. That is the deliberate direction - the alternative is a database
// hiccup turning into "your alerts were silently held" - and a hold is only
// ever applied on a positive answer.
func (s *NotificationDeliveryService) quietUntil(
	ctx context.Context,
	row models.NotificationDelivery,
) (time.Time, bool) {
	if s.quiet == nil {
		return time.Time{}, false
	}
	quiet, until, err := s.quiet.QuietUntil(ctx, row.UserID, time.Now().UTC())
	if err != nil {
		log.Printf("[notification] quiet-hours lookup for user %d failed: %v", row.UserID, err)
		return time.Time{}, false
	}
	if !quiet {
		return time.Time{}, false
	}
	return until, true
}

// holdDelivery puts a claimed row back to pending, due when the quiet period
// ends. `attempts` and `last_status` are deliberately untouched: a hold is the
// sender's decision, not a failed attempt by the endpoint, and counting it
// would eventually park a healthy channel's rows as `failed` for a reason the
// endpoint never caused.
func (s *NotificationDeliveryService) holdDelivery(
	ctx context.Context,
	row models.NotificationDelivery,
	until time.Time,
) {
	if until.IsZero() || !until.After(time.Now().UTC()) {
		// No usable deadline (a window that lapsed between the check and here):
		// come back on the next tick rather than losing the row in the past.
		until = time.Now().UTC().Add(notificationWorkerInterval)
	}
	if err := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ? AND status = ?", row.ID, DeliveryStatusDelivering).
		Updates(map[string]interface{}{
			"status":          DeliveryStatusPending,
			"next_attempt_at": until,
			"held_until":      until,
			"updated_at":      time.Now().UTC(),
		}).Error; err != nil {
		log.Printf("[notification] holding delivery %d failed: %v", row.ID, err)
	}
}

// ReclockHeld moves this user's currently held rows to the new truth after the
// quiet-hours setting changed, and reports how many moved.
//
// Only rows that are *held* are touched - `held_until IS NOT NULL` - so a row
// waiting out a delivery backoff keeps its schedule. That distinction is the
// point of the column: without it, "clear the snooze" would also silently
// shortcut every retry the endpoint was backing off from.
//
// stillHeld selects the two modes: true re-points the hold at a new (earlier)
// quiet-period end, false releases the rows so the worker picks them up now.
func (s *NotificationDeliveryService) ReclockHeld(
	ctx context.Context,
	userID uint,
	at time.Time,
	stillHeld bool,
) (int64, error) {
	at = at.UTC()
	updates := map[string]interface{}{
		"next_attempt_at": at,
		"updated_at":      time.Now().UTC(),
		"held_until":      nil,
	}
	if stillHeld {
		updates["held_until"] = at
	}
	res := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("user_id = ? AND status = ? AND held_until IS NOT NULL AND next_attempt_at > ?",
			userID, DeliveryStatusPending, at).
		Updates(updates)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// claimChannelRow is the single-column projection phase 1 needs: which
// channels have due work. A named type keeps GORM's column mapping explicit.
type claimChannelRow struct {
	ChannelID uint
}

// claimDue atomically flips up to limit due rows to `delivering`. The
// FOR UPDATE SKIP LOCKED claim keeps two workers (or two server instances)
// from sending the same row twice.
//
// The claim is per-channel fair rather than strictly oldest-first. Ordering the
// whole table by id looks right until one endpoint accumulates a backlog: from
// then on every batch is filled by that channel's oldest rows and every other
// channel's notifications sit behind them, no matter how small those channels
// are. So phase 1 picks the channels that hold the oldest due work and phase 2
// takes at most notificationPerChannelBatch rows from each. The batch can
// therefore be shorter than limit when few channels have work; that is the
// deliberate cost of never letting one channel monopolise the worker.
func (s *NotificationDeliveryService) claimDue(ctx context.Context, limit int) ([]models.NotificationDelivery, error) {
	if limit <= 0 {
		return nil, nil
	}
	var claimed []models.NotificationDelivery
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		// Phase 1: which channels have due work, oldest work first. This cannot
		// carry FOR UPDATE - Postgres rejects FOR UPDATE together with the
		// aggregates/window functions a per-channel pick needs - so the locking
		// happens in phase 2, where SKIP LOCKED still guarantees each row goes
		// to exactly one claimer.
		var channelRows []claimChannelRow
		if err := tx.Raw(`SELECT channel_id FROM notification_deliveries
			WHERE status = ? AND next_attempt_at <= ?
			GROUP BY channel_id
			ORDER BY MIN(id)
			LIMIT ?`,
			DeliveryStatusPending, now, limit).Scan(&channelRows).Error; err != nil {
			return err
		}
		for _, ch := range channelRows {
			if len(claimed) >= limit {
				break
			}
			want := min(notificationPerChannelBatch, limit-len(claimed))
			var rows []models.NotificationDelivery
			if err := tx.Raw(`SELECT * FROM notification_deliveries
				WHERE status = ? AND next_attempt_at <= ? AND channel_id = ?
				ORDER BY id
				LIMIT ?
				FOR UPDATE SKIP LOCKED`,
				DeliveryStatusPending, now, ch.ChannelID, want).Scan(&rows).Error; err != nil {
				return err
			}
			claimed = append(claimed, rows...)
		}
		if len(claimed) == 0 {
			return nil
		}
		ids := make([]uint, 0, len(claimed))
		for _, row := range claimed {
			ids = append(ids, row.ID)
		}
		return tx.Model(&models.NotificationDelivery{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":     DeliveryStatusDelivering,
				"updated_at": time.Now().UTC(),
			}).Error
	})
	return claimed, err
}

// deliver performs one HTTP attempt for a claimed row and records the
// outcome: delivered, rescheduled with backoff, or parked as failed.
func (s *NotificationDeliveryService) deliver(ctx context.Context, row models.NotificationDelivery) {
	var channel models.UserNotificationChannel
	if err := s.db.WithContext(ctx).First(&channel, row.ChannelID).Error; err != nil {
		// The channel was deleted while the event was queued: nothing to send.
		s.finishDelivery(ctx, row.ID, 0, DeliveryStatusFailed, 0, "channel no longer exists")
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusFailed, 0, "payload is not valid JSON")
		return
	}
	// A collapsed burst posts the first event's body plus how many occurrences
	// it stands for. The stored payload stays untouched, so the delivery log
	// still shows the event that started the burst.
	if row.OccurrenceCount > 1 {
		payload["aggregated_count"] = row.OccurrenceCount
	}

	status, err := postWebhookOnce(channel.URL, channel.Secret, payload, notificationWebhookHeaders(row.ID, payload))
	attempts := row.Attempts + 1
	now := time.Now().UTC()

	if err == nil && status >= 200 && status < 300 {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusDelivered, status, "")
		return
	}

	message := messageFor(status, err)
	if attempts >= notificationMaxAttempts {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusFailed, status, message)
		return
	}
	if uerr := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ?", row.ID).
		Updates(map[string]interface{}{
			"status":          DeliveryStatusPending,
			"attempts":        attempts,
			"last_status":     status,
			"last_error":      message,
			"next_attempt_at": now.Add(notificationBackoff(attempts)),
			"updated_at":      now,
		}).Error; uerr != nil {
		log.Printf("[notification] reschedule delivery %d failed: %v", row.ID, uerr)
	}
	s.touchChannel(ctx, channel.ID, status, now)
}

// notificationWebhookHeaders is the dedupe contract for an outbox delivery.
// The delivery id is the outbox row, so it survives the retry/backoff loop and
// a manual replay; the event id comes from the payload the subscriber stored
// and lets a receiver collapse the same event arriving on two attempts. Either
// header is omitted when its value is unknown (for example a row queued before
// the event id was added to the payload), so receivers must treat a missing
// header as "no key" rather than as the empty key.
func notificationWebhookHeaders(deliveryID uint, payload map[string]interface{}) map[string]string {
	headers := map[string]string{
		HeaderWebhookDeliveryID: strconv.FormatUint(uint64(deliveryID), 10),
	}
	if eventID, ok := payload["id"].(string); ok && eventID != "" {
		headers[HeaderWebhookEventID] = eventID
	}
	return headers
}

// finishDelivery writes a terminal state and records the outcome on the
// channel so the API can show whether the endpoint is healthy.
func (s *NotificationDeliveryService) finishDelivery(ctx context.Context, id, channelID uint, status string, httpStatus int, message string) {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":      status,
		"last_status": httpStatus,
		"last_error":  message,
		"updated_at":  now,
		"attempts":    gorm.Expr("attempts + 1"),
	}
	if status == DeliveryStatusDelivered {
		updates["delivered_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		log.Printf("[notification] finish delivery %d failed: %v", id, err)
	}
	if channelID != 0 {
		s.touchChannel(ctx, channelID, httpStatus, now)
	}
}

// touchChannel records the last outcome on the channel row.
func (s *NotificationDeliveryService) touchChannel(ctx context.Context, channelID uint, status int, at time.Time) {
	if err := s.db.WithContext(ctx).Model(&models.UserNotificationChannel{}).
		Where("id = ?", channelID).
		Updates(map[string]interface{}{
			"last_triggered": at,
			"last_status":    status,
		}).Error; err != nil {
		log.Printf("[notification] update channel %d status: %v", channelID, err)
	}
}

// ListByChannel returns the delivery log for one channel, newest first.
func (s *NotificationDeliveryService) ListByChannel(ctx context.Context, channelID uint, limit int) ([]models.NotificationDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []models.NotificationDelivery
	err := s.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// Retry re-arms one parked (`failed`) delivery so the worker will try it again
// without waiting for a new event. The attempt budget is reset to zero on
// purpose: a transient outage should not permanently consume a row's tries,
// and the previous failure is still visible in `last_error` until the next
// attempt overwrites it. Rows that are delivered or already in flight are not
// retryable.
func (s *NotificationDeliveryService) Retry(ctx context.Context, channelID, deliveryID uint) (*models.NotificationDelivery, error) {
	var row models.NotificationDelivery
	if err := s.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", deliveryID, channelID).
		First(&row).Error; err != nil {
		return nil, err
	}
	if row.Status != DeliveryStatusFailed {
		return nil, ErrDeliveryNotRetryable
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ?", row.ID).
		Updates(map[string]interface{}{
			"status":          DeliveryStatusPending,
			"attempts":        0,
			"next_attempt_at": now,
			"updated_at":      now,
		}).Error; err != nil {
		return nil, err
	}
	row.Status = DeliveryStatusPending
	row.Attempts = 0
	row.NextAttemptAt = now
	row.UpdatedAt = now
	return &row, nil
}

// UnackedDelivered counts one channel's delivered-but-unconfirmed rows. It is
// the channel-level view of CountAcknowledgements' second number: what did this
// endpoint accept that nobody has confirmed.
func (s *NotificationDeliveryService) UnackedDelivered(ctx context.Context, channelID uint) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&models.NotificationDelivery{}).
		Where("channel_id = ? AND status = ? AND acked_at IS NULL", channelID, DeliveryStatusDelivered).
		Count(&count).Error
	return count, err
}

// MaxRetryFailedLimit bounds RetryFailedFilters.Limit so one request cannot
// arm an unbounded batch.
const MaxRetryFailedLimit = 1000

// Acknowledge records that a person confirmed a finished delivery.
//
// The outbox already knows whether an endpoint accepted the POST, which is a
// weaker fact than it looks: a webhook that answers `200` and then drops the
// event is indistinguishable from one that acted on it. This closes that half -
// "somebody saw it" is a claim only a human or the receiving system acting on
// their behalf can make, so it is recorded separately and never inferred.
//
// Acking is idempotent on purpose. The first ack owns the timestamp; a later
// one returns the same row rather than rewriting history, so a retry by the
// caller (or two people clicking at once) cannot move the clock.
func (s *NotificationDeliveryService) Acknowledge(
	ctx context.Context,
	channelID, deliveryID uint,
) (*models.NotificationDelivery, error) {
	var row models.NotificationDelivery
	if err := s.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", deliveryID, channelID).
		First(&row).Error; err != nil {
		return nil, err
	}
	if row.Status != DeliveryStatusDelivered && row.Status != DeliveryStatusFailed {
		return nil, ErrDeliveryNotConfirmable
	}
	if row.AckedAt != nil {
		return &row, nil
	}
	now := time.Now().UTC()
	// Conditional so two simultaneous acks cannot both claim to be the first:
	// the loser re-reads the winner's timestamp instead of overwriting it.
	if err := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ? AND acked_at IS NULL", row.ID).
		Updates(map[string]interface{}{"acked_at": now, "updated_at": now}).Error; err != nil {
		return nil, err
	}
	// Read the row back rather than returning the in-memory values: a timestamp
	// written with `time.Now().UTC()` serialises as `...Z` while the same column
	// read from the database comes back in the session's zone, and a client that
	// compared the two responses as strings would see a difference that is not
	// there. Every response from this endpoint is now the stored row.
	if err := s.db.WithContext(ctx).First(&row, row.ID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// CountAcknowledgements reports the confirmation loop in two numbers: how many
// deliveries carry a human ack, and how many finished successfully but were
// never confirmed. The second number is the one worth watching - a channel that
// answers 200 into a void shows up there and nowhere else.
func (s *NotificationDeliveryService) CountAcknowledgements(ctx context.Context) (acked int64, unacked int64, err error) {
	if err = s.db.WithContext(ctx).
		Model(&models.NotificationDelivery{}).
		Where("acked_at IS NOT NULL").
		Count(&acked).Error; err != nil {
		return 0, 0, err
	}
	if err = s.db.WithContext(ctx).
		Model(&models.NotificationDelivery{}).
		Where("status = ? AND acked_at IS NULL", DeliveryStatusDelivered).
		Count(&unacked).Error; err != nil {
		return 0, 0, err
	}
	return acked, unacked, nil
}

// MaxRetryFailedChannelIDs bounds RetryFailedFilters.ChannelIDs. The list is a
// narrowing filter, not a bulk-ingest surface, so a handful of endpoints is
// the realistic size and anything much larger is a caller mistake.
const MaxRetryFailedChannelIDs = 50

// MaxRetryFailedSpreadMinutes bounds RetryFailedFilters.SpreadOverMinutes. A
// day is plenty: the point of a spread replay is to smooth a burst, not to
// schedule a backlog for later.
const MaxRetryFailedSpreadMinutes = 1440

// notificationReplaySpreadBuckets is how many due-times a spread replay uses.
// Re-arming every row at once hands the endpoint the whole backlog in one burst
// and can trip a rate limiter that would have coped with the same volume
// spread out; a handful of buckets is enough to avoid that without turning the
// replay into a drip.
const notificationReplaySpreadBuckets = 20

// RetryFailedFilters narrows a bulk replay. The zero value means "every parked
// delivery in scope", which is how the endpoints behaved before filters
// existed.
type RetryFailedFilters struct {
	// EventType limits the replay to one event type, e.g.
	// "device.access.denied".
	EventType string
	// DeviceID limits the replay to deliveries about one device. Rows written
	// before the device_id column existed have an empty value and are never
	// matched by this filter (they still replay without one).
	DeviceID string
	// Limit caps how many rows a single call re-arms, oldest first. 0 means no
	// cap; the handler clamps anything above MaxRetryFailedLimit.
	Limit int
	// SpreadOverMinutes paces the replay across that many minutes instead of
	// making every matched row due at once. 0 keeps the original behaviour
	// (everything due immediately); the handler clamps anything above
	// MaxRetryFailedSpreadMinutes.
	SpreadOverMinutes int
	// ChannelIDs limits the replay to specific channels - the cross-channel
	// endpoint's answer to "one of my five endpoints is broken, leave the
	// others alone". Empty means every channel in scope. On the per-channel
	// endpoint this can only narrow further, never widen.
	ChannelIDs []uint
	// CreatedAfter / CreatedBefore bound the replay by when the event was
	// queued. Both are inclusive. created_at is used rather than updated_at on
	// purpose: updated_at moves on every attempt, so a window over it would
	// mean something different each time it is evaluated.
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// RetryFailedResult reports what a bulk replay matched and what it did.
type RetryFailedResult struct {
	// Matched counts parked rows that satisfy the filters, before Limit.
	Matched int64
	// Requeued counts rows actually re-armed. Always 0 on a dry run.
	Requeued int64
}

// retryFailedQuery builds the "which parked rows are in scope" predicate. It is
// called once per statement (count, then select) because reusing a GORM chain
// after Count can carry statement state.
func retryFailedQuery(ctx context.Context, db *gorm.DB, scopeColumn string, scopeValue uint, f RetryFailedFilters) *gorm.DB {
	q := db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where(scopeColumn+" = ? AND status = ?", scopeValue, DeliveryStatusFailed)
	if et := strings.TrimSpace(f.EventType); et != "" {
		q = q.Where("event_type = ?", et)
	}
	if d := strings.TrimSpace(f.DeviceID); d != "" {
		q = q.Where("device_id = ?", d)
	}
	if len(f.ChannelIDs) > 0 {
		q = q.Where("channel_id IN ?", f.ChannelIDs)
	}
	if f.CreatedAfter != nil {
		q = q.Where("created_at >= ?", f.CreatedAfter.UTC())
	}
	if f.CreatedBefore != nil {
		q = q.Where("created_at <= ?", f.CreatedBefore.UTC())
	}
	return q
}

// RetryFailedForChannel re-arms parked (`failed`) deliveries for one channel.
// It is the bulk form of Retry, for an operator who fixed a flaky endpoint and
// wants the backlog retried without clicking each row.
//
// A dry run counts what would be re-armed and changes nothing, so an operator
// can look before they leap; that matters because re-arming immediately wakes
// the delivery worker, and the obvious fix ("replay everything") is often not
// what the situation calls for.
func (s *NotificationDeliveryService) RetryFailedForChannel(ctx context.Context, channelID uint, f RetryFailedFilters, dryRun bool) (RetryFailedResult, error) {
	return s.retryFailed(ctx, "channel_id", channelID, f, dryRun)
}

// RetryFailedForUser is the cross-channel form: every parked delivery for one
// user, whatever channel it came from. Handy after an outage that took several
// endpoints down at once, and still scoped strictly by user_id so one account
// can never replay another's backlog.
func (s *NotificationDeliveryService) RetryFailedForUser(ctx context.Context, userID uint, f RetryFailedFilters, dryRun bool) (RetryFailedResult, error) {
	if userID == 0 {
		// The whole point of the scope is that a zero value must never turn
		// into "every user's backlog".
		return RetryFailedResult{}, errors.New("user id is required")
	}
	return s.retryFailed(ctx, "user_id", userID, f, dryRun)
}

func (s *NotificationDeliveryService) retryFailed(ctx context.Context, scopeColumn string, scopeValue uint, f RetryFailedFilters, dryRun bool) (RetryFailedResult, error) {
	var out RetryFailedResult
	if err := retryFailedQuery(ctx, s.db, scopeColumn, scopeValue, f).Count(&out.Matched).Error; err != nil {
		return out, err
	}
	if dryRun || out.Matched == 0 {
		return out, nil
	}

	// Oldest first, so a capped replay drains the backlog deterministically
	// instead of re-arming an arbitrary slice each call.
	ids := retryFailedQuery(ctx, s.db, scopeColumn, scopeValue, f).Select("id").Order("id ASC")
	if f.Limit > 0 {
		ids = ids.Limit(f.Limit)
	}
	if f.SpreadOverMinutes > 0 {
		var idList []uint
		if err := ids.Pluck("id", &idList).Error; err != nil {
			return out, err
		}
		return s.rearmSpread(ctx, idList, f.SpreadOverMinutes, out)
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"status":          DeliveryStatusPending,
			"attempts":        0,
			"next_attempt_at": now,
			"updated_at":      now,
		})
	if result.Error != nil {
		return out, result.Error
	}
	out.Requeued = result.RowsAffected
	return out, nil
}

// rearmSpread re-arms rows in staggered buckets so a bulk replay reaches the
// endpoint gradually instead of all at once. The window is nailed to the
// caller's number: the first bucket becomes due after one bucket interval and
// the last at the end of the window, so `spread_over_minutes: N` really does
// take N minutes to work through even though every row is already `pending`.
//
// Each bucket is its own UPDATE. That is deliberate: the status is re-checked
// in every statement, so a row that something else re-armed (or a concurrent
// retry of the same range) is left alone rather than being counted as ours.
func (s *NotificationDeliveryService) rearmSpread(ctx context.Context, idList []uint, spreadMinutes int, out RetryFailedResult) (RetryFailedResult, error) {
	if len(idList) == 0 {
		return out, nil
	}
	buckets := notificationReplaySpreadBuckets
	if len(idList) < buckets {
		buckets = len(idList)
	}
	perBucket := (len(idList) + buckets - 1) / buckets
	step := time.Duration(spreadMinutes) * time.Minute / time.Duration(buckets)
	now := time.Now().UTC()
	var requeued int64
	for start := 0; start < len(idList); start += perBucket {
		end := min(start+perBucket, len(idList))
		bucket := start / perBucket
		due := now.Add(time.Duration(bucket+1) * step)
		result := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
			Where("id IN ? AND status = ?", idList[start:end], DeliveryStatusFailed).
			Updates(map[string]interface{}{
				"status":          DeliveryStatusPending,
				"attempts":        0,
				"next_attempt_at": due,
				"updated_at":      now,
			})
		if result.Error != nil {
			return out, result.Error
		}
		requeued += result.RowsAffected
	}
	out.Requeued = requeued
	return out, nil
}

// RetryAllFailed is RetryFailedForChannel with no filters, kept for callers
// written before filters existed.
func (s *NotificationDeliveryService) RetryAllFailed(ctx context.Context, channelID uint) (int64, error) {
	res, err := s.RetryFailedForChannel(ctx, channelID, RetryFailedFilters{}, false)
	return res.Requeued, err
}

// RetryAllFailedForUser is RetryFailedForUser with no filters.
func (s *NotificationDeliveryService) RetryAllFailedForUser(ctx context.Context, userID uint) (int64, error) {
	res, err := s.RetryFailedForUser(ctx, userID, RetryFailedFilters{}, false)
	return res.Requeued, err
}

// notificationBackoff grows 2s, 4s, 8s … up to the cap.
func notificationBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	backoff := time.Duration(1<<uint(min(attempts, 10))) * time.Second
	if backoff > notificationBackoffCap {
		return notificationBackoffCap
	}
	return backoff
}

func messageFor(status int, err error) string {
	if err != nil {
		return err.Error()
	}
	if status == 0 {
		return "transport error"
	}
	return "HTTP " + strconv.Itoa(status)
}
