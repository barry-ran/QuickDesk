package models

import "time"

type AuditLog struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// AdminID and AdminUsername identify the administrator actor. They stay
	// zero for actions taken by an end user.
	AdminID       uint   `gorm:"index" json:"admin_id"`
	AdminUsername string `gorm:"size:100;index" json:"admin_username"`
	// ActorUserID identifies the end-user actor when the action came from a
	// user rather than an admin. Two columns rather than one "actor" because
	// the two live in different tables and callers filter on one or the other;
	// overloading admin_id with a user id would make "which admin did this?"
	// unanswerable.
	ActorUserID  uint      `gorm:"index;not null;default:0" json:"actor_user_id"`
	Action       string    `gorm:"size:50;index" json:"action"`
	ResourceType string    `gorm:"size:50" json:"resource_type"`
	ResourceID   string    `gorm:"size:100" json:"resource_id"`
	Details      string    `gorm:"type:text" json:"details"`
	IP           string    `gorm:"size:50" json:"ip"`
	CreatedAt    time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
