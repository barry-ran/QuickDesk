package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// ErrInvalidQuietHours means the submitted quiet-hours setting is malformed.
var ErrInvalidQuietHours = errors.New(
	"quiet_hours must be a list of {days 0-6, start/end HH:MM} with a valid IANA timezone")

// MaxQuietHoursWindows bounds how many recurring windows one setting may carry.
const MaxQuietHoursWindows = 20

// quietHoursRecheck is how long a delivery held by a *recurring* window waits
// before asking again. The end of a window is computable, but the evaluator
// that decides "quiet now" does not return it, and re-deriving the boundary
// from {days,start,end} in a second place is how two implementations of one
// rule start disagreeing. A snooze, which is an exact instant, is honoured to
// the second; a recurring window is polled at this granularity and the alert
// is delivered at most this long after the window ends.
const quietHoursRecheck = 5 * time.Minute

// NotificationQuietHoursInput is the validated body of the settings endpoint.
// It is a full replace: omitting the snooze clears it, exactly like the
// offline-alert rule's own snooze.
type NotificationQuietHoursInput struct {
	SnoozedUntil *time.Time
	Windows      []AccessWindow
	Timezone     string
}

// NotificationQuietHoursService owns the per-user quiet-hours setting and
// answers the one question the delivery worker asks: is this user quiet now,
// and until when.
type NotificationQuietHoursService struct {
	db *gorm.DB
}

func NewNotificationQuietHoursService(db *gorm.DB) *NotificationQuietHoursService {
	return &NotificationQuietHoursService{db: db}
}

// Get returns the stored setting, or (nil, nil) when the user never saved one.
// A missing row and a row with nothing set mean the same thing to callers, but
// only the endpoint needs to tell them apart, and it does so by reporting the
// documented defaults either way.
func (s *NotificationQuietHoursService) Get(ctx context.Context, userID uint) (*models.NotificationQuietHours, error) {
	var row models.NotificationQuietHours
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Put replaces the setting. Validation happens before anything is written, so
// a rejected body cannot half-apply.
func (s *NotificationQuietHoursService) Put(
	ctx context.Context,
	userID uint,
	in NotificationQuietHoursInput,
) (*models.NotificationQuietHours, error) {
	if len(in.Windows) > MaxQuietHoursWindows {
		return nil, fmt.Errorf("%w: at most %d windows", ErrInvalidQuietHours, MaxQuietHoursWindows)
	}
	if err := ValidateAccessWindows(in.Windows); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidQuietHours, err)
	}
	if err := ValidateTimezone(in.Timezone); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidQuietHours, err)
	}
	encoded, err := encodeQuietHoursWindows(in.Windows)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := models.NotificationQuietHours{
		UserID:    userID,
		Windows:   encoded,
		Timezone:  strings.TrimSpace(in.Timezone),
		UpdatedAt: now,
	}
	if in.SnoozedUntil != nil {
		row.SnoozedUntil = in.SnoozedUntil
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.NotificationQuietHours
		err := tx.Where("user_id = ?", userID).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			row.CreatedAt = now
			return tx.Create(&row).Error
		case err != nil:
			return err
		}
		updates := map[string]interface{}{
			"snoozed_until": row.SnoozedUntil,
			"windows":       row.Windows,
			"timezone":      row.Timezone,
			"updated_at":    now,
		}
		return tx.Model(&models.NotificationQuietHours{}).
			Where("user_id = ?", userID).
			Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID)
}

// QuietUntil reports whether the user is quiet right now and when that ends.
// The second value is only meaningful when the first is true.
//
// A missing setting is not quiet, which is the behaviour every existing
// deployment has.
func (s *NotificationQuietHoursService) QuietUntil(
	ctx context.Context,
	userID uint,
	now time.Time,
) (bool, time.Time, error) {
	row, err := s.Get(ctx, userID)
	if err != nil {
		return false, time.Time{}, err
	}
	if row == nil {
		return false, time.Time{}, nil
	}
	if until := QuietHoursUntil(*row, now); !until.IsZero() {
		return true, until, nil
	}
	return false, time.Time{}, nil
}

// QuietHoursUntil is the decision itself, separated from storage so the rules
// can be pinned by tests without a database.
func QuietHoursUntil(row models.NotificationQuietHours, now time.Time) time.Time {
	now = now.UTC()
	// A snooze is an exact instant, so a held delivery can be pointed straight
	// at the moment it lifts. It wins over the recurring windows because it is
	// the more specific instruction: "not tonight".
	if row.SnoozedUntil != nil && now.Before(*row.SnoozedUntil) {
		return row.SnoozedUntil.UTC()
	}
	windows := QuietHoursWindows(row.Windows)
	if len(windows) == 0 {
		return time.Time{}
	}
	if active, _ := EvaluateAccessWindowsAt(windows, LocationForPolicy(row.Timezone), now); active {
		return now.Add(quietHoursRecheck)
	}
	return time.Time{}
}

// QuietHoursWindows decodes the stored windows. A malformed row (hand-edited
// database, older schema) decodes to "no windows" so the worker keeps running
// instead of holding every alert forever.
func QuietHoursWindows(stored string) []AccessWindow {
	if strings.TrimSpace(stored) == "" {
		return nil
	}
	var windows []AccessWindow
	if err := json.Unmarshal([]byte(stored), &windows); err != nil {
		return nil
	}
	return windows
}

// encodeQuietHoursWindows stores an empty list as "" rather than "[]" so "no
// windows" has exactly one representation.
func encodeQuietHoursWindows(windows []AccessWindow) (string, error) {
	if len(windows) == 0 {
		return "", nil
	}
	buf, err := json.Marshal(windows)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// NotificationQuietHoursView is the wire shape: the stored fields plus the
// server's own verdict, so a client does not have to reimplement the window
// evaluation (and get the midnight wrap or the timezone wrong).
type NotificationQuietHoursView struct {
	SnoozedUntil *time.Time     `json:"snoozed_until"`
	Windows      []AccessWindow `json:"quiet_hours"`
	Timezone     string         `json:"timezone"`
	Active       bool           `json:"active"`
	// ActiveUntil is when the current quiet period ends: the snooze instant, or
	// the next re-check for a recurring window. Absent when not active.
	ActiveUntil *time.Time `json:"active_until,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// QuietHoursView builds the wire shape. It answers for the stored row *and*
// for "no row at all", because the endpoint reports the same defaults in both
// cases rather than a null the client has to interpret.
func QuietHoursView(row *models.NotificationQuietHours, now time.Time) NotificationQuietHoursView {
	view := NotificationQuietHoursView{Windows: []AccessWindow{}}
	if row == nil {
		return view
	}
	view.SnoozedUntil = row.SnoozedUntil
	view.Timezone = row.Timezone
	if windows := QuietHoursWindows(row.Windows); len(windows) > 0 {
		view.Windows = windows
	}
	updated := row.UpdatedAt.UTC()
	view.UpdatedAt = &updated
	if until := QuietHoursUntil(*row, now); !until.IsZero() {
		view.Active = true
		view.ActiveUntil = &until
	}
	return view
}
