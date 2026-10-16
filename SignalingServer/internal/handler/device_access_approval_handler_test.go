package handler

import (
	"testing"

	"quickdesk/signaling/internal/models"
)

func TestParseApprovalStatusFilter(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   []string
		wantOK bool
	}{
		{"default is pending", "", []string{models.ApprovalPending}, true},
		{"single", "approved", []string{models.ApprovalApproved}, true},
		{"comma list trims blanks", "pending, denied ,", []string{models.ApprovalPending, models.ApprovalDenied}, true},
		{"unknown status rejected", "pending,bogus", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseApprovalStatusFilter(tc.raw)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
