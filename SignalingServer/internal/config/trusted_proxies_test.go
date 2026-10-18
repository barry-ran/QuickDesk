package config

import "testing"

// Empty means "the usual self-hosted topology": the proxy is on this host or on
// the LAN. `none` is the explicit opt-out for a directly exposed server, and a
// list is parsed with the whitespace a human writes into an env file.
func TestParseTrustedProxies(t *testing.T) {
	t.Run("empty falls back to loopback and private ranges", func(t *testing.T) {
		got := parseTrustedProxies("")
		if len(got) != len(DefaultTrustedProxies) {
			t.Fatalf("got %v, want the defaults %v", got, DefaultTrustedProxies)
		}
		// The caller must not be able to mutate the shared default slice.
		got[0] = "203.0.113.0/24"
		if DefaultTrustedProxies[0] == "203.0.113.0/24" {
			t.Fatal("parseTrustedProxies handed out the package default slice")
		}
	})

	t.Run("none trusts nobody", func(t *testing.T) {
		if got := parseTrustedProxies("none"); got != nil {
			t.Fatalf("got %v, want nil (trust nobody)", got)
		}
		if got := parseTrustedProxies("  NONE "); got != nil {
			t.Fatalf("the sentinel should be case and space insensitive, got %v", got)
		}
	})

	t.Run("a list is trimmed", func(t *testing.T) {
		got := parseTrustedProxies(" 10.1.0.0/16 ,127.0.0.1/32, ")
		if len(got) != 2 || got[0] != "10.1.0.0/16" || got[1] != "127.0.0.1/32" {
			t.Fatalf("got %v, want the two trimmed entries", got)
		}
	})
}
