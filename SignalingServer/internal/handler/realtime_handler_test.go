package handler

import (
	"testing"

	"quickdesk/signaling/internal/service"
)

func TestRevokedFamilyIDScopesFamilyBreak(t *testing.T) {
	event := service.Event{
		Type: service.EventSessionRevoked,
		Data: map[string]interface{}{
			"family_id": "family-a",
			"reason":    "family_break",
		},
	}

	if got := revokedFamilyID(event); got != "family-a" {
		t.Fatalf("revokedFamilyID() = %q, want family-a", got)
	}
}

func TestRevokedFamilyIDKeepsGlobalRevocationGlobal(t *testing.T) {
	event := service.Event{
		Type: service.EventSessionRevoked,
		Data: map[string]interface{}{"reason": "password_reset"},
	}

	if got := revokedFamilyID(event); got != "" {
		t.Fatalf("revokedFamilyID() = %q, want empty family for global revocation", got)
	}
}
