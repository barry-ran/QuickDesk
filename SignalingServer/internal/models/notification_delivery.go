package models

import "time"

// NotificationDelivery is one outbox row: a user-scoped event that must reach
// one notification channel. Rows are written when the event happens and are
// retried by the delivery worker until they succeed or run out of attempts,
// so a channel that is briefly down no longer loses the event.
type NotificationDelivery struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ChannelID uint   `gorm:"not null;index" json:"channel_id"`
	UserID    uint   `gorm:"not null;index" json:"user_id"`
	EventType string `gorm:"size:64;not null" json:"event_type"`
	// DeviceID is the device the event was about, when there is one. It is
	// denormalised out of the payload so an operator can replay "just this
	// device's backlog" with an indexed column instead of a JSON scan.
	DeviceID string `gorm:"size:20;index" json:"device_id"`
	Payload  string `gorm:"type:text;not null" json:"payload"` // JSON body actually posted

	// Status is pending | delivering | delivered | failed.
	Status string `gorm:"size:16;not null;index" json:"status"`
	// Attempts counts finished delivery attempts (not claims).
	Attempts int `gorm:"not null;default:0" json:"attempts"`
	// OccurrenceCount is how many matching events this one row represents. It
	// is 1 unless notification aggregation is enabled and a burst of the same
	// (channel, event type, device) collapsed into this row while it was still
	// waiting out its hold. The worker adds it to the posted body as
	// `aggregated_count` when it is greater than 1, so a receiver can tell
	// "this happened once" from "this happened 40 times".
	OccurrenceCount int `gorm:"not null;default:1" json:"occurrence_count"`
	// NextAttemptAt is when the worker may pick the row up again.
	NextAttemptAt time.Time `gorm:"not null;index" json:"next_attempt_at"`

	LastStatus  int        `json:"last_status"` // HTTP status of the last attempt, 0 = transport error
	LastError   string     `gorm:"size:255" json:"last_error"`
	DeliveredAt *time.Time `json:"delivered_at"`
	// AckedAt is when a human confirmed the delivery. It is deliberately
	// separate from DeliveredAt: "the endpoint accepted the POST" (2xx) and
	// "somebody actually looked at this" are different facts, and a webhook
	// that answers 200 into a void is exactly the failure this column makes
	// visible. Exactly one of the two can be set without the other.
	AckedAt *time.Time `json:"acked_at"`
	// HeldUntil, when set, means the row is currently parked by the owner's
	// quiet hours rather than waiting on a backoff: it will not be attempted
	// before that instant. It is cleared when the hold is released, so it is a
	// current fact and not a history - the deliveries list can therefore answer
	// "why is this alert still pending" without guessing from timestamps.
	HeldUntil *time.Time `json:"held_until"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (NotificationDelivery) TableName() string {
	return "notification_deliveries"
}
