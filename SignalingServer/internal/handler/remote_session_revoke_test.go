package handler

import (
	"net/http/httptest"
	"testing"

	"quickdesk/signaling/internal/models"

	"github.com/gin-gonic/gin"
)

// A revoked socket has to be distinguishable from one the network dropped, so
// the reason is recorded on the connection and read by the deferred close. The
// one rule worth pinning is that the first writer wins: a revoke that arrives a
// moment after the socket already died must not relabel a disconnect as an
// operator action.
func TestSignalConnEndReasonIsFirstWriterWins(t *testing.T) {
	var sc signalConn
	if got := sc.endReasonOrDefault(models.RemoteSessionEndDisconnected); got != models.RemoteSessionEndDisconnected {
		t.Fatalf("an unmarked connection defaulted to %q, want %q", got, models.RemoteSessionEndDisconnected)
	}
	sc.markEndReason(models.RemoteSessionEndRevoked)
	sc.markEndReason(models.RemoteSessionEndDisconnected)
	if got := sc.endReasonOrDefault(models.RemoteSessionEndDisconnected); got != models.RemoteSessionEndRevoked {
		t.Fatalf("the first reason was overwritten: got %q, want %q", got, models.RemoteSessionEndRevoked)
	}
	// An empty reason is not a reason.
	var blank signalConn
	blank.markEndReason("")
	blank.markEndReason("  ")
	if got := blank.endReasonOrDefault(models.RemoteSessionEndDisconnected); got != models.RemoteSessionEndDisconnected {
		t.Fatalf("blank marks changed the reason: got %q", got)
	}
}

// The revoke path finds the socket by session id. The lookup must not confuse
// two controllers on one device, and it must not match on an empty id.
func TestFindSessionConnMatchesBySessionID(t *testing.T) {
	wanted := &signalConn{sessionID: "sig_wanted"}
	h := &RealtimeHandler{signalClient: map[string]map[string]*signalConn{
		"111111111": {
			"cli-a": &signalConn{sessionID: "sig_other"},
			"cli-b": wanted,
		},
		"222222222": {
			"cli-c": &signalConn{sessionID: "sig_far"},
		},
	}}
	if got := h.findSessionConn("sig_wanted"); got != wanted {
		t.Fatalf("findSessionConn returned %p, want the connection for sig_wanted", got)
	}
	if got := h.findSessionConn("sig_missing"); got != nil {
		t.Fatalf("findSessionConn returned %p for an unknown session, want nil", got)
	}
	if got := h.findSessionConn("  "); got != nil {
		t.Fatalf("findSessionConn matched a blank session id: %p", got)
	}
}

// fakeRevoker records what the shared revoke body asked of the socket topology.
type fakeRevoker struct {
	called  bool
	session string
	found   bool
}

func (f *fakeRevoker) RevokeSession(sessionID string) bool {
	f.called = true
	f.session = sessionID
	return f.found
}

func newRevokeTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "/v1/me/remote-sessions/sig_test/revoke", nil)
	return c, rec
}

// A session that is already over is a 409, not a success that did nothing - and
// the socket layer must not even be consulted.
func TestRevokeSessionSocketRefusesAFinishedSession(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	rev := &fakeRevoker{found: true}
	row := &models.RemoteSession{SessionID: "sig_test", Active: false}
	if revokeSessionSocket(c, rev, row) {
		t.Fatal("revokeSessionSocket reported success for a session that was already over")
	}
	if rev.called {
		t.Fatal("a finished session still reached the socket layer")
	}
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

// A live row whose socket is not on this instance cannot be closed from here;
// the caller gets a 409 rather than a hollow 200.
func TestRevokeSessionSocketRefusesWhenTheSocketIsNotHere(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	rev := &fakeRevoker{found: false}
	row := &models.RemoteSession{SessionID: "sig_test", Active: true}
	if revokeSessionSocket(c, rev, row) {
		t.Fatal("revokeSessionSocket reported success with no live socket")
	}
	if !rev.called || rev.session != "sig_test" {
		t.Fatalf("the socket layer was not asked about this session: %+v", rev)
	}
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

// The happy path closes the socket and leaves the response to the caller.
func TestRevokeSessionSocketClosesALiveSession(t *testing.T) {
	c, _ := newRevokeTestContext(t)
	rev := &fakeRevoker{found: true}
	row := &models.RemoteSession{SessionID: "sig_test", Active: true}
	if !revokeSessionSocket(c, rev, row) {
		t.Fatal("revokeSessionSocket refused a live session that was connected here")
	}
	if !rev.called || rev.session != "sig_test" {
		t.Fatalf("the socket layer was asked about the wrong session: %+v", rev)
	}
}

// A handler built without a revoker must say so instead of panicking.
func TestRevokeSessionSocketWithoutARevoker(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	row := &models.RemoteSession{SessionID: "sig_test", Active: true}
	if revokeSessionSocket(c, nil, row) {
		t.Fatal("revokeSessionSocket succeeded with no revoker configured")
	}
	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
