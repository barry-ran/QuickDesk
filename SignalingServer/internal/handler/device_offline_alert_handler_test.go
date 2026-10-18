package handler

import (
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"
)

// A client should be able to render the offline-alert form before the owner has
// ever saved anything, so "no rule" has to come back as the documented defaults
// rather than an empty object or a 404.
func TestOfflineAlertViewDefaultsWithoutRule(t *testing.T) {
	view := offlineAlertView("123456789", nil)
	if view.DeviceID != "123456789" {
		t.Fatalf("device_id = %q, want the requested device", view.DeviceID)
	}
	if view.Enabled {
		t.Fatal("a device with no rule must not report an enabled alert")
	}
	if view.AfterMinutes != service.DefaultOfflineAlertAfterMinutes {
		t.Fatalf("after_minutes = %d, want the default %d",
			view.AfterMinutes, service.DefaultOfflineAlertAfterMinutes)
	}
	if view.RepeatMinutes != 0 {
		t.Fatalf("repeat_minutes = %d, want 0 (one alert per outage)", view.RepeatMinutes)
	}
	if view.OfflineSince != nil || view.LastAlertedAt != nil || view.SnoozedUntil != nil || view.UpdatedAt != nil {
		t.Fatal("a device with no rule must not report episode state")
	}
}

// The episode state is exposed on purpose: it is how an owner understands why
// an alert has or has not fired yet.
func TestOfflineAlertViewCarriesEpisodeState(t *testing.T) {
	since := time.Now().UTC().Add(-11 * time.Minute)
	alerted := since.Add(10 * time.Minute)
	rule := &models.DeviceOfflineAlertRule{
		DeviceID:     "123456789",
		Enabled:      true,
		AfterMinutes: 10,
		OfflineSince: &since,
		AlertedAt:    &alerted,
		UpdatedAt:    alerted,
	}
	view := offlineAlertView("123456789", rule)
	if !view.Enabled || view.AfterMinutes != 10 {
		t.Fatalf("view = %+v, want enabled with a 10 minute threshold", view)
	}
	if view.OfflineSince == nil || !view.OfflineSince.Equal(since) {
		t.Fatalf("offline_since = %v, want %v", view.OfflineSince, since)
	}
	if view.LastAlertedAt == nil || !view.LastAlertedAt.Equal(alerted) {
		t.Fatalf("last_alerted_at = %v, want %v", view.LastAlertedAt, alerted)
	}
}
