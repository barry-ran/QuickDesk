package models

import "time"

// InstanceLogEntry is one log line an instance shipped into the shared
// database, so an admin search can cover every running instance instead of only
// the machine that happened to answer the request.
//
// The row is a *copy*. The authoritative log stays the rotating file on the
// instance that wrote it, which is why aggregation can be dropped under
// pressure without losing the line.
type InstanceLogEntry struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// InstanceID names the process the line came from. It is the same id the
	// server logs at startup and the presence service registers, so a line can
	// be traced back to a concrete process.
	InstanceID string `gorm:"size:64;not null;uniqueIndex:idx_instance_log_seq,priority:1" json:"instance_id"`
	// Seq is the shipping sink's own per-instance counter. Together with the
	// instance and the source it is the idempotency key: a batch that is sent
	// twice (a retry after a timeout that had actually committed) inserts the
	// same keys and is ignored rather than duplicated.
	Seq int64 `gorm:"not null;uniqueIndex:idx_instance_log_seq,priority:3" json:"seq"`
	// Source is the logical log source the line came from; today the
	// application log, but a sink could ship the PostgreSQL or Redis log too.
	Source string `gorm:"size:64;not null;default:'';uniqueIndex:idx_instance_log_seq,priority:2" json:"source"`
	// LoggedAt is the timestamp parsed from the line's own prefix, falling back
	// to the moment the sink received it when the line has none.
	LoggedAt  time.Time `gorm:"not null;index" json:"logged_at"`
	Line      string    `gorm:"type:text;not null" json:"line"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName overrides the default table name.
func (InstanceLogEntry) TableName() string {
	return "instance_log_entries"
}
