package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNotificationEventPayloadCarriesEventID pins the dedupe contract: every
// user notification body carries the event envelope id so a receiver has a
// stable key even when the event data has no unique field of its own. An empty
// id is omitted rather than posted as "" so a receiver never keys on a blank
// value.
func TestNotificationEventPayloadCarriesEventID(t *testing.T) {
	payload := notificationEventPayload(Event{
		ID:   "evt_abc",
		Type: EventDeviceAccessDenied,
		Ts:   time.Unix(0, 0).UTC(),
		Data: map[string]interface{}{"reason": "invalid_code"},
	})
	if got := payload["id"]; got != "evt_abc" {
		t.Errorf("payload id = %v, want evt_abc", got)
	}
	if got := payload["event"]; got != EventDeviceAccessDenied {
		t.Errorf("payload event = %v, want %s", got, EventDeviceAccessDenied)
	}

	withoutID := notificationEventPayload(Event{Type: EventDeviceBound, Ts: time.Unix(0, 0).UTC()})
	if _, ok := withoutID["id"]; ok {
		t.Error("an event with no id must not post an empty id field")
	}
}

// TestNotificationWebhookHeaders checks the header pair a receiver dedupes on.
func TestNotificationWebhookHeaders(t *testing.T) {
	headers := notificationWebhookHeaders(42, map[string]interface{}{"id": "evt_abc"})
	if got := headers[HeaderWebhookDeliveryID]; got != "42" {
		t.Errorf("delivery id header = %q, want 42", got)
	}
	if got := headers[HeaderWebhookEventID]; got != "evt_abc" {
		t.Errorf("event id header = %q, want evt_abc", got)
	}

	// A row queued before the event id existed, or a payload whose id is not a
	// string, must drop the event header instead of sending a junk value.
	for _, payload := range []map[string]interface{}{
		{},
		{"id": ""},
		{"id": 12345},
	} {
		got := notificationWebhookHeaders(7, payload)
		if got[HeaderWebhookDeliveryID] != "7" {
			t.Errorf("delivery id header missing for payload %v", payload)
		}
		if _, ok := got[HeaderWebhookEventID]; ok {
			t.Errorf("event id header must be omitted for payload %v, got %q", payload, got[HeaderWebhookEventID])
		}
	}
}

// TestPostWebhookOnceSendsIdentityHeaders proves the headers actually reach the
// wire, and that a caller-supplied header cannot replace the HMAC signature the
// receiver verifies.
func TestPostWebhookOnceSendsIdentityHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := notificationEventPayload(Event{
		ID:   "evt_wire",
		Type: EventDeviceAccessDenied,
		Ts:   time.Now().UTC(),
	})
	headers := notificationWebhookHeaders(99, payload)
	// A caller trying to spoof the signature must lose to the real one.
	headers["X-Webhook-Signature"] = "sha256=deadbeef"

	status, err := postWebhookOnce(srv.URL, "s3cret", payload, headers)
	if err != nil {
		t.Fatalf("postWebhookOnce: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if got.Get(HeaderWebhookDeliveryID) != "99" {
		t.Errorf("delivery id header = %q, want 99", got.Get(HeaderWebhookDeliveryID))
	}
	if got.Get(HeaderWebhookEventID) != "evt_wire" {
		t.Errorf("event id header = %q, want evt_wire", got.Get(HeaderWebhookEventID))
	}
	sig := got.Get("X-Webhook-Signature")
	if !strings.HasPrefix(sig, "sha256=") || sig == "sha256=deadbeef" {
		t.Errorf("signature header = %q, want the real digest to win", sig)
	}
}
