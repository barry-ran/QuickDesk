package service

import (
	"context"
	"errors"
	"testing"
)

func TestIsDeviceAccessResult(t *testing.T) {
	tests := []struct {
		result string
		want   bool
	}{
		{DeviceAccessGranted, true},
		{DeviceAccessDeniedInvalidCode, true},
		{DeviceAccessDeniedRateLimited, true},
		{DeviceAccessHostOffline, true},
		{DeviceAccessSessionStarted, true},
		{DeviceAccessDeniedPolicy, true},
		{DeviceAccessDeniedSessionLimit, true},
		{"", false},
		{"denied", false},
		{"GRANTED", false},
	}
	for _, tt := range tests {
		if got := IsDeviceAccessResult(tt.result); got != tt.want {
			t.Errorf("IsDeviceAccessResult(%q) = %v, want %v", tt.result, got, tt.want)
		}
	}
}

// Record must reject bad input before it touches the database: the callers
// are on the access-code verification path and pass a nil-db service here,
// so a regression that reorders the checks panics instead of failing quietly.
func TestRecordDeviceAccessRejectsBadInput(t *testing.T) {
	svc := &DeviceAccessService{}
	ctx := context.Background()

	if _, err := svc.Record(ctx, RecordDeviceAccessInput{
		DeviceID: "123456789",
		Result:   "not-a-result",
	}); !errors.Is(err, ErrInvalidDeviceAccessResult) {
		t.Errorf("invalid result: err = %v, want ErrInvalidDeviceAccessResult", err)
	}

	if _, err := svc.Record(ctx, RecordDeviceAccessInput{
		DeviceID: "   ",
		Result:   DeviceAccessGranted,
	}); err == nil {
		t.Error("blank device_id: want error, got nil")
	}
}
