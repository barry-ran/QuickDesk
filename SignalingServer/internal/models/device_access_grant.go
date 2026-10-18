package models

import "time"

// DeviceAccessGrant is a time-boxed, optionally single-use code that admits a
// controller to one device without handing over the device's own (rotating)
// access code. The owner creates it; VerifyAccessCode accepts either the device
// code or an active grant.
//
// Grants are deliberately additive to the device policy, not a bypass: the
// policy gate (remote switch, IP allow list, windows, exception dates) runs
// before the code comparison, so a grant can never open a device that the owner
// has switched off.
type DeviceAccessGrant struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DeviceID string `gorm:"size:20;not null;index" json:"device_id"`
	// OwnerID is the user who issued the grant (audit value; unrelated to the
	// guest who redeems it).
	OwnerID uint `gorm:"index" json:"owner_id"`
	// Code is what the guest types. It is shown to the owner on creation (and
	// in the list) because the owner has to communicate it.
	Code  string `gorm:"size:64;not null;index" json:"code"`
	Label string `gorm:"size:100" json:"label"`
	// ExpiresAt bounds the grant in time; a grant past this instant is refused.
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	// MaxUses caps how many sessions the grant may start. 0 means "no cap
	// within the window".
	MaxUses int `gorm:"not null;default:0" json:"max_uses"`
	// Uses counts sessions actually started with the grant.
	Uses int `gorm:"not null;default:0" json:"uses"`
	// RevokedAt lets the owner cancel a grant before it expires.
	RevokedAt  *time.Time `json:"revoked_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	// BoundClientID, when set, restricts the grant to the controller that
	// presents this client_id. client_id is self-asserted by the client, so
	// this is friction against an accidental leak rather than proof of
	// identity - the server cannot verify who a controller really is today.
	BoundClientID string `gorm:"size:64" json:"bound_client_id"`
	// BoundClientPublicKey turns the client-id binding from an assertion into a
	// proof when it is set: the controller must sign the server's challenge
	// with the matching private key, so a leaked code plus a leaked client id
	// is no longer enough. It holds the base64 (standard alphabet) encoding of
	// a raw 32-byte Ed25519 public key, and empty keeps the old behaviour.
	BoundClientPublicKey string `gorm:"size:100" json:"bound_client_public_key"`
	// BoundIPs is a JSON array of IPs/CIDRs, also optional. Unlike the client
	// id this one is server-observed, so it is an actual boundary: a leaked
	// code stays useless from anywhere else. Empty means "any address".
	BoundIPs string `gorm:"type:text" json:"bound_ips"`
	// LastUsedClientID / LastUsedIP record who actually redeemed the grant, so
	// the owner can see how a code was used rather than only that it was.
	LastUsedClientID string `gorm:"size:64" json:"last_used_client_id"`
	LastUsedIP       string `gorm:"size:50" json:"last_used_ip"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (DeviceAccessGrant) TableName() string {
	return "device_access_grants"
}

// UsableAt reports whether the grant may admit a controller at the given time.
func (g DeviceAccessGrant) UsableAt(now time.Time) bool {
	if g.RevokedAt != nil {
		return false
	}
	if !now.Before(g.ExpiresAt) {
		return false
	}
	return g.MaxUses <= 0 || g.Uses < g.MaxUses
}
