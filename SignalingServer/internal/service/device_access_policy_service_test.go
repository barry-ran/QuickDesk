package service

import (
	"errors"
	"testing"

	"quickdesk/signaling/internal/models"
)

func TestValidateAllowIPs(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		wantErr bool
	}{
		{"empty list allows anything", nil, false},
		{"exact ipv4", []string{"10.211.55.4"}, false},
		{"ipv4 cidr", []string{"10.211.55.0/24"}, false},
		{"ipv6 and cidr", []string{"::1", "2001:db8::/32"}, false},
		{"trimmed entry", []string{"  10.0.0.1  "}, false},
		{"hostname is not enough", []string{"example.com"}, true},
		{"bad cidr", []string{"10.0.0.0/99"}, true},
		{"empty entry", []string{"  "}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAllowIPs(tt.entries)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAllowIPs(%v) err = %v, wantErr %v", tt.entries, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidAllowIP) {
				t.Fatalf("error should wrap ErrInvalidAllowIP, got %v", err)
			}
		})
	}
}

func TestIPAllowed(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		ip      string
		want    bool
	}{
		{"empty list allows", nil, "127.0.0.1", true},
		{"exact match", []string{"127.0.0.1"}, "127.0.0.1", true},
		{"exact mismatch", []string{"127.0.0.1"}, "10.0.0.1", false},
		{"cidr match", []string{"10.211.55.0/24"}, "10.211.55.4", true},
		{"cidr mismatch", []string{"10.211.55.0/24"}, "10.211.56.4", false},
		{"one of many matches", []string{"192.168.0.0/16", "10.211.55.4"}, "10.211.55.4", true},
		{"unparsable client ip fails closed", []string{"10.0.0.0/8"}, "not-an-ip", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IPAllowed(tt.entries, tt.ip); got != tt.want {
				t.Fatalf("IPAllowed(%v, %q) = %v, want %v", tt.entries, tt.ip, got, tt.want)
			}
		})
	}
}

func TestEvaluateDeviceAccessPolicy(t *testing.T) {
	// No policy row at all: every existing device keeps working.
	if allowed, reason := EvaluateDeviceAccessPolicy(nil, "127.0.0.1"); !allowed || reason != "" {
		t.Fatalf("nil policy should allow, got allowed=%v reason=%q", allowed, reason)
	}

	disabled := &models.DeviceAccessPolicy{RemoteDisabled: true, AllowIPs: `["127.0.0.1"]`}
	if allowed, reason := EvaluateDeviceAccessPolicy(disabled, "127.0.0.1"); allowed {
		t.Fatal("remote_disabled must block even an allow-listed IP")
	} else if reason == "" {
		t.Fatal("expected a reason for the blocked attempt")
	}

	allowListed := &models.DeviceAccessPolicy{AllowIPs: `["10.211.55.0/24"]`}
	if allowed, _ := EvaluateDeviceAccessPolicy(allowListed, "10.211.55.4"); !allowed {
		t.Fatal("client inside the allow list should pass")
	}
	if allowed, reason := EvaluateDeviceAccessPolicy(allowListed, "127.0.0.1"); allowed {
		t.Fatal("client outside the allow list must be blocked")
	} else if reason == "" {
		t.Fatal("expected a reason for the blocked attempt")
	}

	emptyList := &models.DeviceAccessPolicy{AllowIPs: `[]`}
	if allowed, _ := EvaluateDeviceAccessPolicy(emptyList, "8.8.8.8"); !allowed {
		t.Fatal("an empty allow list means any IP")
	}
}

func TestSessionLimitReached(t *testing.T) {
	tests := []struct {
		name   string
		max    int
		active int
		want   bool
	}{
		{"unlimited (0)", 0, 9, false},
		{"negative treated as unlimited", -3, 9, false},
		{"limit 1 with nobody connected", 1, 0, false},
		{"limit 1 with one connected", 1, 1, true},
		{"limit 2 with one connected", 2, 1, false},
		{"limit 2 with two connected", 2, 2, true},
		{"already above the limit", 2, 5, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SessionLimitReached(tt.max, tt.active); got != tt.want {
				t.Fatalf("SessionLimitReached(%d, %d) = %v, want %v", tt.max, tt.active, got, tt.want)
			}
		})
	}
}

// The approval switch started as an all-or-nothing boolean; modes let the
// owner target the sessions that are actually risky (a guest code, an
// unfamiliar address) instead of approving every login from their own laptop.
func TestDecideApprovalModes(t *testing.T) {
	policy := func(mode, trusted string) *models.DeviceAccessPolicy {
		return &models.DeviceAccessPolicy{ApprovalMode: mode, ApprovalTrustedIPs: trusted}
	}
	const (
		deviceCode = models.ApprovalSourceDeviceCode
		grant      = models.ApprovalSourceGrant
	)

	tests := []struct {
		name     string
		policy   *models.DeviceAccessPolicy
		source   string
		ip       string
		want     bool
		wantTrig string
	}{
		{"nil policy never asks", nil, deviceCode, "203.0.113.9", false, ""},
		{"off never asks", policy(ApprovalModeOff, ""), grant, "203.0.113.9", false, ""},
		{"always asks for the device code", policy(ApprovalModeAlways, ""), deviceCode, "10.0.0.1", true, ApprovalTriggerAlways},
		{"always asks for a grant", policy(ApprovalModeAlways, ""), grant, "10.0.0.1", true, ApprovalTriggerAlways},
		{"grant mode ignores the device code", policy(ApprovalModeGrant, ""), deviceCode, "203.0.113.9", false, ""},
		{"grant mode asks for a guest code", policy(ApprovalModeGrant, ""), grant, "203.0.113.9", true, ApprovalTriggerGrant},
		{"untrusted_ip exempts a trusted address", policy(ApprovalModeUntrustedIP, `["10.211.55.0/24"]`), deviceCode, "10.211.55.4", false, ""},
		{"untrusted_ip asks for an outside address", policy(ApprovalModeUntrustedIP, `["10.211.55.0/24"]`), deviceCode, "203.0.113.9", true, ApprovalTriggerUntrustedIP},
		{"untrusted_ip asks for a grant from outside", policy(ApprovalModeUntrustedIP, `["10.211.55.0/24"]`), grant, "203.0.113.9", true, ApprovalTriggerUntrustedIP},
		// A hand-edited row with no trusted addresses must fail closed rather
		// than treat "empty list" as "trust everyone".
		{"untrusted_ip with no trusted list fails closed", policy(ApprovalModeUntrustedIP, `[]`), deviceCode, "10.0.0.1", true, ApprovalTriggerUntrustedIP},
		{"unparseable client ip is untrusted", policy(ApprovalModeUntrustedIP, `["10.0.0.0/8"]`), deviceCode, "", true, ApprovalTriggerUntrustedIP},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideApproval(tt.policy, tt.source, tt.ip)
			if got.Required != tt.want {
				t.Fatalf("Required = %v, want %v", got.Required, tt.want)
			}
			if got.Trigger != tt.wantTrig {
				t.Fatalf("Trigger = %q, want %q", got.Trigger, tt.wantTrig)
			}
		})
	}
}

// The original boolean must keep working after modes were introduced, so a
// policy written by an older client (or an existing row) is never silently
// downgraded to "no approval".
func TestNormalizeApprovalModeKeepsLegacyBoolean(t *testing.T) {
	cases := []struct {
		mode   string
		legacy bool
		want   string
	}{
		{"", false, ApprovalModeOff},
		{"", true, ApprovalModeAlways},
		{"always", true, ApprovalModeAlways},
		{"off", true, ApprovalModeOff},
		{" GRANT ", false, ApprovalModeGrant},
		{"untrusted_ip", false, ApprovalModeUntrustedIP},
		{"nonsense", true, ApprovalModeAlways},
		{"nonsense", false, ApprovalModeOff},
	}
	for _, tc := range cases {
		if got := NormalizeApprovalMode(tc.mode, tc.legacy); got != tc.want {
			t.Fatalf("NormalizeApprovalMode(%q, %v) = %q, want %q", tc.mode, tc.legacy, got, tc.want)
		}
	}
	if got := EffectiveApprovalMode(nil); got != ApprovalModeOff {
		t.Fatalf("EffectiveApprovalMode(nil) = %q, want off", got)
	}
}

func TestValidateApprovalMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		trusted []string
		wantErr bool
	}{
		{"off", ApprovalModeOff, nil, false},
		{"always", ApprovalModeAlways, nil, false},
		{"grant", ApprovalModeGrant, nil, false},
		{"untrusted_ip with a CIDR", ApprovalModeUntrustedIP, []string{"10.211.55.0/24"}, false},
		{"untrusted_ip with an exact IP", ApprovalModeUntrustedIP, []string{"10.0.0.1"}, false},
		{"untrusted_ip with an empty list", ApprovalModeUntrustedIP, nil, true},
		{"untrusted_ip with blank entries only", ApprovalModeUntrustedIP, []string{"  "}, true},
		{"untrusted_ip with junk", ApprovalModeUntrustedIP, []string{"example.com"}, true},
		{"unknown mode", "maybe", nil, true},
		{"bad trusted ip on any mode", ApprovalModeAlways, []string{"not-an-ip"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateApprovalMode(tt.mode, tt.trusted)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateApprovalMode(%q, %v) = %v, wantErr %v", tt.mode, tt.trusted, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrInvalidApprovalMode) {
				t.Fatalf("error should wrap ErrInvalidApprovalMode, got %v", err)
			}
		})
	}
}

// A typo in approval_mode must be a 400, not a silent "off": that particular
// fallback would disable the very gate the caller asked to turn on. This caught
// a live failure where {"approval_mode":"maybe"} was accepted and stored as
// "off".
func TestParseApprovalModeInputRejectsTypos(t *testing.T) {
	ok := []struct {
		mode   string
		legacy bool
		want   string
	}{
		{"", false, ApprovalModeOff},
		{"", true, ApprovalModeAlways},
		{"off", true, ApprovalModeOff},
		{"always", false, ApprovalModeAlways},
		{"grant", false, ApprovalModeGrant},
		{"untrusted_ip", false, ApprovalModeUntrustedIP},
		{"  GRANT  ", false, ApprovalModeGrant},
		{"grant ", false, ApprovalModeGrant}, // surrounding space is tolerated
	}
	for _, tc := range ok {
		got, err := ParseApprovalModeInput(tc.mode, tc.legacy)
		if err != nil || got != tc.want {
			t.Fatalf("ParseApprovalModeInput(%q, %v) = (%q, %v), want %q", tc.mode, tc.legacy, got, err, tc.want)
		}
	}
	for _, bad := range []string{"maybe", "alwyas", "on", "true", "1"} {
		if got, err := ParseApprovalModeInput(bad, true); err == nil {
			t.Fatalf("ParseApprovalModeInput(%q) = %q with no error, want rejection", bad, got)
		} else if !errors.Is(err, ErrInvalidApprovalMode) {
			t.Fatalf("ParseApprovalModeInput(%q) error should wrap ErrInvalidApprovalMode, got %v", bad, err)
		}
	}
}
