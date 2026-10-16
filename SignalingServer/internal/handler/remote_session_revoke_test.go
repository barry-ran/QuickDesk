package handler

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"quickdesk/signaling/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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

// The bus is addressed, not broadcast: a command must land only on the channel
// named for the instance that owns the session, or every instance would try to
// close a socket it does not have.
func TestSessionRevokeBusChannelIsAddressedAtOneInstance(t *testing.T) {
	b := NewSessionRevokeBus(&redis.Client{}, "inst-a", nil)
	if got, want := b.channel("inst-a"), "qd:sessions:instance:inst-a:revoke"; got != want {
		t.Fatalf("channel = %q, want %q", got, want)
	}
	if b.channel("inst-a") == b.channel("inst-b") {
		t.Fatal("two instances share one revoke channel")
	}
	for _, tc := range []struct{ instance, session string }{
		{"", "sig_x"},
		{"inst-a", ""},
		{"  ", "  "},
	} {
		if err := b.Publish(context.Background(), tc.instance, tc.session); err == nil {
			t.Fatalf("Publish(%q, %q) was accepted", tc.instance, tc.session)
		}
	}
}

// fakeRevoker records what the shared revoke body asked of the socket layer.
type fakeRevoker struct {
	called     bool
	session    string
	found      bool
	asked      bool
	askedInst  string
	askedSess  string
	askErr     error
	instanceID string
}

func (f *fakeRevoker) RevokeSession(sessionID string) bool {
	f.called = true
	f.session = sessionID
	return f.found
}

func (f *fakeRevoker) AskInstanceRevoke(_ context.Context, instanceID, sessionID string) error {
	f.asked = true
	f.askedInst = instanceID
	f.askedSess = sessionID
	return f.askErr
}

func (f *fakeRevoker) InstanceID() string { return f.instanceID }

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
	if revokeSessionSocket(c, nil, rev, row) {
		t.Fatal("revokeSessionSocket reported success for a session that was already over")
	}
	if rev.called || rev.asked {
		t.Fatal("a finished session still reached the socket layer")
	}
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

// A live row with no recorded instance cannot be attributed to anybody, so it is
// a 409 rather than a broadcast to the whole fleet.
func TestRevokeSessionSocketRefusesARowWithNoInstance(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	rev := &fakeRevoker{found: false, instanceID: "inst-here"}
	row := &models.RemoteSession{SessionID: "sig_test", Active: true}
	if revokeSessionSocket(c, nil, rev, row) {
		t.Fatal("revokeSessionSocket reported success with no recorded instance")
	}
	if rev.asked {
		t.Fatal("a row with no instance id was still published to the fleet")
	}
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

// A row that names *this* instance but whose socket we cannot find means the
// socket is already gone; asking ourselves would loop forever.
func TestRevokeSessionSocketDoesNotAskItself(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	rev := &fakeRevoker{found: false, instanceID: "inst-here"}
	row := &models.RemoteSession{SessionID: "sig_test", Active: true, InstanceID: "inst-here"}
	if revokeSessionSocket(c, nil, rev, row) {
		t.Fatal("revokeSessionSocket reported success for a socket that is not here")
	}
	if rev.asked {
		t.Fatal("the revoke was published back to this same instance")
	}
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

// The cross-instance path: the command is addressed at the owning instance, and
// because nothing confirms the row closed, the answer is still a 409 - never a
// success for a publish that nobody acted on.
func TestRevokeSessionSocketAsksTheOwningInstanceAndWaits(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	rev := &fakeRevoker{found: false, instanceID: "inst-here"}
	row := &models.RemoteSession{SessionID: "sig_test", Active: true, InstanceID: "inst-far"}
	if revokeSessionSocket(c, nil, rev, row) {
		t.Fatal("revokeSessionSocket reported success without the row closing")
	}
	if !rev.asked || rev.askedInst != "inst-far" || rev.askedSess != "sig_test" {
		t.Fatalf("the command was not addressed at the owning instance: %+v", rev)
	}
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

// A publish that fails is a server-side problem, not a conflict: the caller
// should see that the command never left.
func TestRevokeSessionSocketReportsAPublishFailure(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	rev := &fakeRevoker{found: false, instanceID: "inst-here", askErr: errors.New("redis is down")}
	row := &models.RemoteSession{SessionID: "sig_test", Active: true, InstanceID: "inst-far"}
	if revokeSessionSocket(c, nil, rev, row) {
		t.Fatal("revokeSessionSocket reported success after a failed publish")
	}
	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

// The happy path closes the socket on this instance and leaves the response to
// the caller.
func TestRevokeSessionSocketClosesALiveSession(t *testing.T) {
	c, _ := newRevokeTestContext(t)
	rev := &fakeRevoker{found: true, instanceID: "inst-here"}
	row := &models.RemoteSession{SessionID: "sig_test", Active: true, InstanceID: "inst-here"}
	if !revokeSessionSocket(c, nil, rev, row) {
		t.Fatal("revokeSessionSocket refused a live session that was connected here")
	}
	if !rev.called || rev.session != "sig_test" {
		t.Fatalf("the socket layer was asked about the wrong session: %+v", rev)
	}
	if rev.asked {
		t.Fatal("a local revoke was also published to another instance")
	}
}

// A handler built without a revoker must say so instead of panicking.
func TestRevokeSessionSocketWithoutARevoker(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	row := &models.RemoteSession{SessionID: "sig_test", Active: true}
	if revokeSessionSocket(c, nil, nil, row) {
		t.Fatal("revokeSessionSocket succeeded with no revoker configured")
	}
	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
