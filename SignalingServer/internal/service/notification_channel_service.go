package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SupportedUserNotificationEvents are the user-scoped events a user-owned
// channel may subscribe to. The list is explicit on purpose: admin webhooks
// only carry system-scope events (§2.17 M4), so anything tied to one user's
// devices has to be opted into here, and a typo in an event name should fail
// loudly at create time instead of silently never firing.
var SupportedUserNotificationEvents = []string{
	EventDeviceAccessDenied,
	EventDeviceAccessGranted,
	EventDeviceAccessSession,
	EventDeviceOnlineChanged,
	EventDeviceOfflineSustained,
	EventDeviceBound,
	EventDeviceUnbound,
	EventDeviceAccessApprovalRequested,
	EventDeviceAccessApprovalDecided,
	// A scheduled task that has spent its retry budget is an operational
	// failure the owner asked to be told about by scheduling it.
	EventScheduledTaskFailed,
}

// IsSupportedUserNotificationEvent reports whether evtType may be subscribed
// to by a user notification channel.
func IsSupportedUserNotificationEvent(evtType string) bool {
	for _, e := range SupportedUserNotificationEvents {
		if e == evtType {
			return true
		}
	}
	return false
}

var (
	// ErrInvalidNotificationURL means the endpoint is not a usable http(s) URL.
	ErrInvalidNotificationURL = errors.New("notification URL must be an absolute http(s) URL")
	// ErrUnsupportedNotificationEvent means the caller subscribed to an event
	// outside SupportedUserNotificationEvents.
	ErrUnsupportedNotificationEvent = errors.New("unsupported notification event")
)

// ValidateNotificationURL rejects anything that is not an absolute http(s)
// endpoint. Note for operators: this is not an SSRF guard by itself — a
// self-hosted deployment still lets a user point a channel at an internal
// address, which is acceptable for a single-tenant install and should be
// revisited before this is exposed as multi-tenant SaaS.
func ValidateNotificationURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ErrInvalidNotificationURL
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return ErrInvalidNotificationURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidNotificationURL
	}
	if u.Host == "" {
		return ErrInvalidNotificationURL
	}
	return nil
}

// NotificationChannelService owns per-user webhook endpoints: CRUD plus the
// delivery path that turns user-scoped bus events into outbound HTTP calls.
type NotificationChannelService struct {
	db         *gorm.DB
	deliveries *NotificationDeliveryService
}

func NewNotificationChannelService(db *gorm.DB, deliveries *NotificationDeliveryService) *NotificationChannelService {
	return &NotificationChannelService{db: db, deliveries: deliveries}
}

// UpsertNotificationChannelInput is the validated body of the create/update
// endpoints.
type UpsertNotificationChannelInput struct {
	Name    string
	URL     string
	Secret  string
	Events  []string
	Enabled bool
}

func (s *NotificationChannelService) Create(ctx context.Context, userID uint, in UpsertNotificationChannelInput) (*models.UserNotificationChannel, error) {
	if err := validateChannelInput(in); err != nil {
		return nil, err
	}
	eventsJSON, _ := json.Marshal(in.Events)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "channel"
	}
	row := &models.UserNotificationChannel{
		UserID:  userID,
		Name:    name,
		URL:     strings.TrimSpace(in.URL),
		Secret:  in.Secret,
		Events:  string(eventsJSON),
		Enabled: in.Enabled,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *NotificationChannelService) List(ctx context.Context, userID uint) ([]models.UserNotificationChannel, error) {
	var rows []models.UserNotificationChannel
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&rows).Error
	return rows, err
}

func (s *NotificationChannelService) Get(ctx context.Context, userID, id uint) (*models.UserNotificationChannel, error) {
	var row models.UserNotificationChannel
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *NotificationChannelService) Update(ctx context.Context, userID, id uint, in UpsertNotificationChannelInput) (*models.UserNotificationChannel, error) {
	if err := validateChannelInput(in); err != nil {
		return nil, err
	}
	row, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	eventsJSON, _ := json.Marshal(in.Events)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "channel"
	}
	row.Name = name
	row.URL = strings.TrimSpace(in.URL)
	if in.Secret != "" {
		row.Secret = in.Secret
	}
	row.Events = string(eventsJSON)
	row.Enabled = in.Enabled
	if err := s.db.WithContext(ctx).Save(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *NotificationChannelService) Delete(ctx context.Context, userID, id uint) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.UserNotificationChannel{}).Error
}

// DispatchEvent fans one user-scoped event out to that user's matching
// channels. Called from the event bus subscriber, so it must not block: each
// delivery runs in its own goroutine and only the delivery itself writes the
// last_triggered / last_status bookkeeping.
func (s *NotificationChannelService) DispatchEvent(ctx context.Context, evt Event) {
	if evt.UserID == 0 || !IsSupportedUserNotificationEvent(evt.Type) {
		return
	}
	var channels []models.UserNotificationChannel
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND enabled = ?", evt.UserID, true).
		Find(&channels).Error; err != nil {
		log.Printf("[notification] load channels for user %d: %v", evt.UserID, err)
		return
	}
	if len(channels) == 0 {
		return
	}
	payload := notificationEventPayload(evt)
	for _, ch := range channels {
		if !channelSubscribes(ch.Events, evt.Type) {
			continue
		}
		if s.deliveries == nil {
			log.Printf("[notification] no delivery queue configured; dropping %s for channel %d",
				evt.Type, ch.ID)
			continue
		}
		if err := s.deliveries.Enqueue(ctx, ch, evt.Type, evt.DeviceID, payload); err != nil {
			log.Printf("[notification] enqueue %s for channel %d failed: %v", evt.Type, ch.ID, err)
		}
	}
}

// notificationEventPayload is the body posted to a user's channel. The top
// level `id` is the event envelope's stable identity: delivery is
// at-least-once, so a receiver needs a value it can key a dedupe table on even
// when the event data itself carries no unique field. It is echoed back in the
// X-Webhook-Event-Id header on the outbox POST.
func notificationEventPayload(evt Event) map[string]interface{} {
	payload := map[string]interface{}{
		"event":     evt.Type,
		"timestamp": evt.Ts.UTC().Format(time.RFC3339),
		"data":      evt.Data,
	}
	if evt.ID != "" {
		payload["id"] = evt.ID
	}
	return payload
}

// SendTest delivers a synthetic `test` event so a user can verify their
// endpoint without waiting for real device traffic.
func (s *NotificationChannelService) SendTest(ctx context.Context, userID, id uint) error {
	row, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	payload := notificationEventPayload(Event{
		ID:   "evt_" + uuid.New().String(),
		Type: "test",
		Ts:   time.Now().UTC(),
		Data: map[string]interface{}{
			"message": "This is a test notification from QuickDesk",
		},
	})
	if s.deliveries == nil {
		return errors.New("delivery queue is not configured")
	}
	return s.deliveries.Enqueue(ctx, *row, "test", "", payload)
}

func validateChannelInput(in UpsertNotificationChannelInput) error {
	if err := ValidateNotificationURL(in.URL); err != nil {
		return err
	}
	if len(in.Events) == 0 {
		return fmt.Errorf("%w: at least one event is required", ErrUnsupportedNotificationEvent)
	}
	for _, e := range in.Events {
		if !IsSupportedUserNotificationEvent(e) {
			return fmt.Errorf("%w: %q (supported: %s)",
				ErrUnsupportedNotificationEvent, e, strings.Join(SupportedUserNotificationEvents, ", "))
		}
	}
	return nil
}

// channelSubscribes reports whether the stored JSON event list contains
// evtType. A malformed list never matches.
func channelSubscribes(eventsJSON, evtType string) bool {
	var events []string
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		return false
	}
	for _, e := range events {
		if e == evtType {
			return true
		}
	}
	return false
}
