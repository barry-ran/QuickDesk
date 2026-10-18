package service

import (
	"encoding/json"
	"strings"
	"testing"

	"quickdesk/signaling/internal/models"
)

func TestValidateNotificationURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://example.com/hook", false},
		{"http://127.0.0.1:9000/notify", false},
		{"  https://example.com/hook  ", false},
		{"", true},
		{"example.com/hook", true}, // no scheme
		{"ftp://example.com/hook", true},
		{"file:///etc/passwd", true},
		{"https://", true}, // no host
	}
	for _, tt := range tests {
		err := ValidateNotificationURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateNotificationURL(%q) err = %v, wantErr %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestIsSupportedUserNotificationEvent(t *testing.T) {
	for _, evt := range SupportedUserNotificationEvents {
		if !IsSupportedUserNotificationEvent(evt) {
			t.Errorf("event %q should be subscribable", evt)
		}
	}
	// System-scope events belong to admin webhooks, not user channels.
	for _, evt := range []string{EventTurnConfigChang, EventSessionRevoked, "nope"} {
		if IsSupportedUserNotificationEvent(evt) {
			t.Errorf("event %q must not be subscribable by a user channel", evt)
		}
	}
}

func TestChannelSubscribes(t *testing.T) {
	list := `["device.access.denied","device.online.changed"]`
	if !channelSubscribes(list, EventDeviceAccessDenied) {
		t.Error("expected device.access.denied to match")
	}
	if channelSubscribes(list, EventDeviceAccessSession) {
		t.Error("expected session_started NOT to match")
	}
	if channelSubscribes("not json", EventDeviceAccessDenied) {
		t.Error("malformed event list must never match")
	}
	if channelSubscribes("", EventDeviceAccessDenied) {
		t.Error("empty event list must never match")
	}
}

// The signing secret must never leave the server: the API returns raw model
// rows, so json:"-" is the guard.
func TestUserNotificationChannelHidesSecret(t *testing.T) {
	row := models.UserNotificationChannel{
		ID:     1,
		UserID: 7,
		Name:   "ops",
		URL:    "https://example.com/hook",
		Secret: "super-secret",
		Events: `["device.access.denied"]`,
	}
	payload, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(payload), "super-secret") || strings.Contains(string(payload), "secret") {
		t.Fatalf("channel JSON leaks the secret: %s", payload)
	}
}
