package service

import (
	"reflect"
	"testing"
)

// The full-text mode has to stay a different tool from `q=`: words and phrases
// are lexemes, not substrings. These cases mirror the aggregated-store test so
// the file scanner answers the same questions.
func TestLogMatchQuery(t *testing.T) {
	cases := []struct {
		name string
		expr string
		line string
		want bool
	}{
		{"a word matches a whole lexeme", "refused", "connection refused by peer", true},
		{"a word does not match inside a lexeme", "refused", "connection refusedsomething", false},
		{"two words are ANDed", "connection refused", "connection refused by peer", true},
		{"two words apart satisfy the AND", "connection refused", "refused later, connection dropped", true},
		{"a quoted phrase needs adjacency", `"connection refused"`, "connection refused by peer", true},
		{"a quoted phrase rejects the words apart", `"connection refused"`, "refused later, connection dropped", false},
		{"negation excludes the term", "connection -refused", "connection refused by peer", false},
		{"negation keeps the rest", "connection -refused", "connection reset by peer", true},
		{"or alternates", "refused or reset", "connection reset by peer", true},
		{"or still needs one side", "refused or reset", "connection dropped", false},
		{"a leading or is a word", "or foo", "or foo", true},
		{"a trailing or is a word", "foo or", "foo and or", true},
		{"an underscore is a phrase separator", "api_auth", "component=api_auth event=ok", true},
		{"an underscore term needs the parts adjacent", "api_auth", "api then auth", false},
		{"a bare dash contributes nothing", "-", "anything at all", false},
		{"matching is case-insensitive", "CONNECTION", "connection refused", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLogMatch(tc.expr).matches(tc.line)
			if got != tc.want {
				t.Fatalf("match %q against %q = %v, want %v", tc.expr, tc.line, got, tc.want)
			}
		})
	}
}

// The lexer keeps the parts of Postgres's `simple` configuration that appear in
// real log lines: lowercasing, alphanumeric runs as one lexeme, decimals and
// signed numbers.
func TestLogLexemes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"component=api_auth event=request", []string{"component", "api", "auth", "event", "request"}},
		{"abc123 v1.2 12.5ms qd1", []string{"abc123", "v1.2", "12.5", "ms", "qd1"}},
		{"2026-10-16T12:00:00Z", []string{"2026", "-10", "-16", "t12", "00", "00z"}},
		{"error=\"connection refused\"", []string{"error", "connection", "refused"}},
	}
	for _, tc := range cases {
		if got := logLexemes(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("logLexemes(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
