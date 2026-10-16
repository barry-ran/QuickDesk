package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// whoamiRouter mirrors the one decision that matters in main.go: which peers
// are allowed to set X-Forwarded-For.
func whoamiRouter(t *testing.T, trusted []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(trusted); err != nil {
		t.Fatalf("SetTrustedProxies(%v): %v", trusted, err)
	}
	router.GET("/whoami", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})
	return router
}

func clientIPFor(t *testing.T, router *gin.Engine, remoteAddr, forwardedFor string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Body.String()
}

// The whole point of the trusted-proxy list: a header from a peer that is not a
// proxy is a claim, not a fact. Before this was configured, gin believed the
// header from anyone, so allow_ips, approval_trusted_ips, the per-IP rate
// limits and the access log all recorded whatever the client typed.
func TestClientIPIgnoresForwardedForFromUntrustedPeer(t *testing.T) {
	router := whoamiRouter(t, []string{"127.0.0.1/32", "::1/128"})

	if got := clientIPFor(t, router, "203.0.113.9:41234", "198.51.100.77"); got != "203.0.113.9" {
		t.Fatalf("client ip = %q, want the socket address 203.0.113.9", got)
	}

	// A real proxy on this host is believed, which is the deployment the
	// default list exists for.
	if got := clientIPFor(t, router, "127.0.0.1:41234", "198.51.100.77"); got != "198.51.100.77" {
		t.Fatalf("client ip = %q, want the forwarded address from a trusted peer", got)
	}

	// `none` means nobody is a proxy - not even loopback.
	strict := whoamiRouter(t, nil)
	if got := clientIPFor(t, strict, "127.0.0.1:41234", "198.51.100.77"); got != "127.0.0.1" {
		t.Fatalf("client ip = %q, want the socket address when no proxy is trusted", got)
	}
}

// A multi-hop chain is the case the single-value test cannot reach, and it is
// where the policy actually matters: each proxy appends to the header, so only
// the entries near the server are trustworthy, and a client that prepends a
// fake address must not be believed.
func TestClientIPWalksMultiHopForwardedFor(t *testing.T) {
	trusted := []string{"127.0.0.1/32", "::1/128", "10.0.0.0/8", "192.168.0.0/16"}
	router := whoamiRouter(t, trusted)

	cases := []struct {
		name         string
		remoteAddr   string
		forwardedFor string
		want         string
	}{
		{
			name:         "trusted inner proxy, real client leftmost",
			remoteAddr:   "127.0.0.1:41234",
			forwardedFor: "198.51.100.7, 10.0.0.9",
			want:         "198.51.100.7",
		},
		{
			name:         "a prepended fake address is ignored",
			remoteAddr:   "127.0.0.1:41234",
			forwardedFor: "10.0.0.250, 203.0.113.9",
			want:         "203.0.113.9",
		},
		{
			name:         "untrusted rightmost hop wins",
			remoteAddr:   "127.0.0.1:41234",
			forwardedFor: "198.51.100.7, 203.0.113.9",
			want:         "203.0.113.9",
		},
		{
			name:         "all hops trusted falls back to the leftmost",
			remoteAddr:   "127.0.0.1:41234",
			forwardedFor: "10.0.0.1, 192.168.1.5",
			want:         "10.0.0.1",
		},
		{
			name:         "a malformed entry falls back to the socket address",
			remoteAddr:   "127.0.0.1:41234",
			forwardedFor: "198.51.100.7, not-an-ip",
			want:         "127.0.0.1",
		},
		{
			name:         "a chain from an untrusted peer is ignored entirely",
			remoteAddr:   "203.0.113.9:41234",
			forwardedFor: "198.51.100.7, 10.0.0.9",
			want:         "203.0.113.9",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clientIPFor(t, router, tc.remoteAddr, tc.forwardedFor); got != tc.want {
				t.Fatalf("client ip = %q, want %q", got, tc.want)
			}
		})
	}
}
