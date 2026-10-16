package models

import "time"

// Approval statuses. They are persisted and shown to both the requesting
// client and the device owner, so keep them stable.
const (
	// ApprovalPending is waiting for the owner to decide.
	ApprovalPending = "pending"
	// ApprovalApproved was accepted by the owner and may be claimed once.
	ApprovalApproved = "approved"
	// ApprovalDenied was refused by the owner.
	ApprovalDenied = "denied"
	// ApprovalExpired ran past ExpiresAt without a decision.
	ApprovalExpired = "expired"
	// ApprovalConsumed was claimed: the controller got its signal token.
	ApprovalConsumed = "consumed"
	// ApprovalGranted is the status the controller sees on the *answer* to
	// `access-code:verify` and to the claim endpoint when a signal token was
	// minted. It is deliberately not one of the row's lifecycle states: the row
	// says "consumed", the answer says "here is your token".
	ApprovalGranted = "granted"
)

// Access approval sources: which credential the owner is being asked about.
const (
	ApprovalSourceDeviceCode = "device_code"
	ApprovalSourceGrant      = "grant"
)

// DeviceAccessApproval is a second-factor gate on top of a matched access
// code (or a time-boxed grant): the code is correct, but the session only
// starts after the device owner approves it. It is the "someone is asking to
// get onto my machine right now" knob, not a replacement for the access code.
//
// A row is created by `access-code:verify` when the device policy has
// ApprovalRequired set and everything else already passed. The requesting
// client gets a one-time secret (stored here as a SHA-256 hash) to poll/claim
// with; the owner decides through the `/v1/me/...` endpoints.
type DeviceAccessApproval struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DeviceID string `gorm:"size:9;not null;index" json:"device_id"`
	// OwnerID is the device owner who must decide (0 when the device is
	// unbound: nobody can approve, so the request just expires).
	OwnerID uint `gorm:"index" json:"owner_id"`
	// ClientID/IP describe the requesting controller, same as the access log.
	ClientID string `gorm:"size:64" json:"client_id"`
	ClientIP string `gorm:"size:50" json:"client_ip"`
	// Source is ApprovalSourceDeviceCode or ApprovalSourceGrant.
	Source string `gorm:"size:16;not null;default:device_code" json:"source"`
	// GrantID is set when Source is a grant, so the claim path can consume it
	// only once the session can actually start.
	GrantID *uint `gorm:"index" json:"grant_id"`
	// SecretHash is the SHA-256 (hex) of the one-time claim secret. The
	// plaintext is handed to the client and never stored.
	SecretHash string `gorm:"size:64;not null" json:"-"`
	// Status is one of the Approval* constants.
	Status string `gorm:"size:16;not null;index;default:pending" json:"status"`
	// ExpiresAt bounds how long the owner has to decide.
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`

	DecidedAt *time.Time `json:"decided_at"`
	DecidedBy uint       `json:"decided_by"`
	Reason    string     `gorm:"size:255" json:"reason"`
	// ConsumedAt is set when a controller actually exchanged the approval for
	// a signal token, making the row single-use.
	ConsumedAt *time.Time `json:"consumed_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (DeviceAccessApproval) TableName() string {
	return "device_access_approvals"
}

// EffectiveStatusAt collapses a still-pending row past its deadline into
// `expired` without requiring a write. A decision is never un-made: a row the
// owner already approved or denied keeps that status even past ExpiresAt.
func (a DeviceAccessApproval) EffectiveStatusAt(now time.Time) string {
	if a.Status == ApprovalPending && !now.Before(a.ExpiresAt) {
		return ApprovalExpired
	}
	return a.Status
}

// IsTerminal reports whether the row can no longer change without a new
// request.
func (a DeviceAccessApproval) IsTerminal(now time.Time) bool {
	switch a.EffectiveStatusAt(now) {
	case ApprovalDenied, ApprovalExpired, ApprovalConsumed:
		return true
	default:
		return false
	}
}
