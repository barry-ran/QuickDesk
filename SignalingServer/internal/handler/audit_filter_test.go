package handler

import "testing"

// The actor filter is the only way to ask "what did this end user do", because
// user-side rows leave admin_id zero. A malformed value must be an error rather
// than a filter that quietly matches everything.
func TestParseActorUserID(t *testing.T) {
	ok := []struct {
		raw  string
		want uint
	}{
		{"", 0},
		{"   ", 0},
		{"0", 0},
		{"7", 7},
		{" 42 ", 42},
	}
	for _, tc := range ok {
		got, err := parseActorUserID(tc.raw)
		if err != nil {
			t.Errorf("parseActorUserID(%q) errored: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseActorUserID(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}

	for _, raw := range []string{"abc", "-1", "1.5", "7x"} {
		if _, err := parseActorUserID(raw); err == nil {
			t.Errorf("parseActorUserID(%q) accepted a malformed value", raw)
		}
	}
}
