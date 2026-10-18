package models

import "time"

// DeviceAccessPolicy is the device owner's control over *who may open a
// session on this machine*, enforced at the same choke point that writes the
// access log (`access-code:verify`). The access log answers "who tried"; this
// answers "who is allowed".
//
// One row per device; a missing row means "no policy" (remote access allowed
// from anywhere), so existing devices keep working after the upgrade.
type DeviceAccessPolicy struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DeviceID string `gorm:"size:9;not null;uniqueIndex" json:"device_id"`
	// OwnerID is the user who last changed the policy; kept for audit value
	// when a device changes hands.
	OwnerID uint `gorm:"index" json:"owner_id"`
	// RemoteDisabled blocks every new session regardless of the access code.
	RemoteDisabled bool `gorm:"not null;default:false" json:"remote_disabled"`
	// AllowIPs is a JSON array of exact IPs and/or CIDR blocks. Empty means
	// "any client IP". Entries are matched against the controller's address
	// as seen by the server.
	AllowIPs string `gorm:"type:text" json:"allow_ips"`
	// MaxSessions caps how many controllers may hold a signaling session on
	// this device at the same time. 0 means unlimited. Enforced when a client
	// authenticates its signaling socket, i.e. after the access code was
	// already accepted, so the log shows granted + denied_session_limit.
	MaxSessions int `gorm:"not null;default:0" json:"max_sessions"`
	// AllowedWindows is a JSON array of recurring access windows
	// (`[{"days":[1,2,3,4,5],"start":"09:00","end":"18:00"}]`). Empty means
	// "any time". Enforced on verify, so a controller outside the window
	// cannot even guess access codes.
	AllowedWindows string `gorm:"type:text" json:"allowed_windows"`
	// Timezone is the IANA zone the windows are evaluated in (e.g.
	// "Asia/Shanghai"). Empty means the server's local zone.
	Timezone string `gorm:"size:64" json:"timezone"`
	// ExceptionDates is a JSON array of one-off date overrides
	// (`[{"date":"2026-02-17","mode":"deny"}]`) evaluated in Timezone: "deny"
	// blocks the whole day, "allow" permits it and overrides AllowedWindows.
	// Empty means "no exceptions".
	ExceptionDates string `gorm:"type:text" json:"exception_dates"`
	// ApprovalRequired turns a matched access code (or grant) into a pending
	// request the owner must approve before the controller gets a signal
	// token. It is a second factor on top of the code, never a replacement:
	// the code is still compared first, and the policy gate above still runs.
	//
	// Kept for wire compatibility; ApprovalMode is the real switch. When
	// ApprovalMode is empty, ApprovalRequired=true still means "always".
	ApprovalRequired bool `gorm:"not null;default:false" json:"approval_required"`
	// ApprovalMode decides *when* a matched code needs the owner's yes:
	//   ""             - fall back to ApprovalRequired (off / always);
	//   "off"          - never require approval;
	//   "always"       - every session, whatever the credential;
	//   "grant"        - only sessions admitted with a time-boxed grant, i.e.
	//                    the guest code, never the device's own code;
	//   "untrusted_ip" - only clients outside ApprovalTrustedIPs, i.e. the
	//                    "someone from somewhere new" case.
	ApprovalMode string `gorm:"size:16;not null;default:''" json:"approval_mode"`
	// ApprovalTrustedIPs is a JSON array of IPs/CIDRs that skip approval when
	// ApprovalMode is "untrusted_ip". Required (non-empty) in that mode.
	ApprovalTrustedIPs string `gorm:"type:text" json:"approval_trusted_ips"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (DeviceAccessPolicy) TableName() string {
	return "device_access_policies"
}
