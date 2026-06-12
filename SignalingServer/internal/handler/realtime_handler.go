package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RealtimeHandler owns the two WebSocket endpoints:
//
//   GET /v1/realtime/events  鈥?per-user push stream of device.*, favorite.*,
//                              session.revoked, etc. First client frame is
//                              auth; first server frame is auth_ok + snapshot.
//                              Supports {type:"resume", since_rev:N}.
//
//   GET /v1/realtime/signal  鈥?generic signaling WebSocket. First frame
//                              supplies a single-use signal_token plus role
//                              (host / client) and device_id. Messages after
//                              auth_ok are jingle (SDP / ICE) passed opaquely
//                              between host and clients of the same device.
//
// All upgrades happen without any auth in the URL 鈥?all tokens live in the
// first frame (搂2.13).
type RealtimeHandler struct {
	upgrader websocket.Upgrader

	tokens    *service.TokenService
	bus       *service.EventBus
	presence  *service.PresenceService
	db        *gorm.DB
	devices   *service.DeviceService
	favorites *service.FavoriteService
	metrics   *service.MetricsService
	rdb       *redis.Client

	// Per-user fan-out for realtime events.
	eventsMu sync.RWMutex
	events   map[uint]map[*eventConn]struct{}

	// Signaling topology: for each device_id, the single host connection
	// (if any) + a set of client connections keyed by client_id.
	signalMu     sync.RWMutex
	signalHosts  map[string]*signalConn
	signalClient map[string]map[string]*signalConn // device_id -> client_id -> conn
}

func NewRealtimeHandler(
	tokens *service.TokenService,
	bus *service.EventBus,
	presence *service.PresenceService,
	db *gorm.DB,
	devices *service.DeviceService,
	favorites *service.FavoriteService,
	metrics *service.MetricsService,
	rdb *redis.Client,
) *RealtimeHandler {
	h := &RealtimeHandler{
		tokens:    tokens,
		bus:       bus,
		presence:  presence,
		db:        db,
		devices:   devices,
		favorites: favorites,
		metrics:   metrics,
		rdb:       rdb,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				// CORS enforcement is done at the HTTP middleware layer;
				// once the upgrade succeeds we've already validated the
				// origin. The WS itself is authenticated via first-frame
				// auth so this callback just waves the handshake through.
				return true
			},
		},
		events:       map[uint]map[*eventConn]struct{}{},
		signalHosts:  map[string]*signalConn{},
		signalClient: map[string]map[string]*signalConn{},
	}
	// Subscribe to the event bus so user-scoped events fan out to
	// connected events WebSockets.
	bus.Subscribe(h)
	return h
}

// Name implements service.EventSubscriber.
func (h *RealtimeHandler) Name() string { return "realtime" }

// HandleEvent implements service.EventSubscriber: fan out to every WS for
// the event's target user. System-scope events (UserID == 0) are ignored
// here — they go to webhook/audit subscribers instead (§2.17 M5).
//
// `evt.ServerRev` is already populated by EventBus.Publish (and stored on
// the per-user stream), so the rev that goes out on the wire matches what
// resume/snapshot replay sees.
func (h *RealtimeHandler) HandleEvent(ctx context.Context, evt service.Event) {
	if evt.UserID == 0 {
		return
	}

	// For session.revoked we must only notify the connection(s) that belong
	// to the revoked family — NOT every WS for this user. Broadcasting to
	// all connections would log out other devices (e.g. Qt client) that
	// share the same user_id but have their own independent session family.
	revokedFamily := ""
	if evt.Type == service.EventSessionRevoked {
		if fid, ok := evt.Data["family_id"]; ok {
			revokedFamily, _ = fid.(string)
		}
	}

	h.eventsMu.RLock()
	conns := h.events[evt.UserID]
	// Copy to avoid holding the lock while writing.
	snap := make([]*eventConn, 0, len(conns))
	for c := range conns {
		if revokedFamily != "" && c.familyID != revokedFamily {
			continue // skip connections that belong to a different session
		}
		snap = append(snap, c)
	}
	h.eventsMu.RUnlock()
	for _, c := range snap {
		c.send(eventWire{
			ID:        evt.ID,
			Type:      evt.Type,
			Ts:        evt.Ts,
			ServerRev: evt.ServerRev,
			Data:      evt.Data,
		})
	}
}

// nextRev returns the next monotonically-increasing server_rev. Used only
// for the `auth_ok` envelope to give the client an initial checkpoint;
// real events get their rev stamped centrally inside EventBus.Publish so
// stream and live fanout stay in lock-step.
func (h *RealtimeHandler) nextRev() int64 {
	if h.rdb == nil {
		return 0
	}
	v, err := h.rdb.Incr(context.Background(), "qd:events:server_rev").Result()
	if err != nil {
		log.Printf("[realtime] INCR server_rev failed: %v", err)
		return 0
	}
	return v
}

// =====================================================================
// /v1/realtime/events
// =====================================================================

type eventConn struct {
	h        *RealtimeHandler
	conn     *websocket.Conn
	userID   uint
	familyID string // refresh-token family this WS authenticated with
	out      chan eventWire
	done     chan struct{}
	once     sync.Once
}

// eventWire is the canonical events envelope (搂2.7).
type eventWire struct {
	ID        string                 `json:"id,omitempty"`
	Type      string                 `json:"type"`
	Ts        time.Time              `json:"ts,omitempty"`
	ServerRev int64                  `json:"server_rev,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// authFrame is what every WS sends as its first message.
type authFrame struct {
	Type        string `json:"type"`
	AccessToken string `json:"access_token,omitempty"`
	SinceRev    int64  `json:"since_rev,omitempty"`

	// Signal-role fields:
	SignalToken string `json:"signal_token,omitempty"`
	Role        string `json:"role,omitempty"`
	DeviceID    string `json:"device_id,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
}

const (
	firstFrameTimeout = 5 * time.Second
	wsWriteTimeout    = 10 * time.Second
	wsPingInterval    = 5 * time.Second
	wsReadTimeout     = 15 * time.Second
)

// HandleEvents serves GET /v1/realtime/events.
func (h *RealtimeHandler) HandleEvents(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // upgrader has already written the error
	}
	// Read the auth frame with a hard timeout (搂2.13).
	conn.SetReadDeadline(time.Now().Add(firstFrameTimeout))
	var af authFrame
	if err := conn.ReadJSON(&af); err != nil {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(4401, "auth timeout"), time.Now().Add(time.Second))
		_ = conn.Close()
		return
	}
	if af.Type != "auth" || af.AccessToken == "" {
		_ = conn.WriteJSON(eventWire{Type: "error", Data: map[string]interface{}{"code": "AUTH_INVALID"}})
		_ = conn.Close()
		return
	}
	familyID, uid, err := h.tokens.LookupAccessToken(c.Request.Context(), service.ScopeUser, af.AccessToken)
	if err != nil {
		_ = conn.WriteJSON(eventWire{Type: "error", Data: map[string]interface{}{"code": "TOKEN_INVALID"}})
		_ = conn.Close()
		return
	}
	conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
		return nil
	})

	ec := &eventConn{
		h:        h,
		conn:     conn,
		userID:   uid,
		familyID: familyID,
		out:      make(chan eventWire, 64),
		done:     make(chan struct{}),
	}

	// §2.8 requires the bootstrap sequence (auth_ok → snapshot → replay)
	// to be *contiguous* — no concurrent event may be interleaved. We
	// therefore send the bootstrap synchronously BEFORE registering this
	// conn with the bus fanout. Bus Publish can't find us yet, so it
	// can't race us into writing a mid-stream event before snapshot.
	rev := h.nextRev()
	if err := conn.WriteJSON(eventWire{Type: "auth_ok", ServerRev: rev}); err != nil {
		return
	}

	// Replay-then-snapshot: when the client asks to resume, try the
	// stream first — if the gap can be replayed exactly, skip the full
	// snapshot. Otherwise we send `snapshot_required` so the client
	// knows it must rebuild local state from the snapshot frame.
	resumed := false
	if af.SinceRev > 0 {
		if r, err := h.bus.EventsSinceRev(c.Request.Context(), uid, af.SinceRev); err == nil && !r.Truncated {
			for _, e := range r.Events {
				_ = conn.WriteJSON(eventWire{
					ID:        e.ID,
					Type:      e.Type,
					Ts:        e.Ts,
					ServerRev: e.ServerRev,
					Data:      e.Data,
				})
			}
			resumed = true
		} else {
			// Either the stream was trimmed past the caller's
			// checkpoint or the lookup failed; both mean "we cannot
			// guarantee a clean replay" → fall through to the full
			// snapshot path with an explicit hint frame.
			_ = conn.WriteJSON(eventWire{Type: "snapshot_required", ServerRev: rev})
		}
	}
	if !resumed {
		snap, err := h.buildSnapshot(c.Request.Context(), uid)
		if err == nil {
			_ = conn.WriteJSON(eventWire{Type: "snapshot", ServerRev: rev, Data: snap})
		}
	}

	// Now that the bootstrap is fully flushed, make this conn visible to
	// the bus fanout and start the ping/read loops.
	h.registerEventConn(ec)
	defer h.unregisterEventConn(ec)

	go ec.writer()
	ec.reader()
}

func (h *RealtimeHandler) registerEventConn(ec *eventConn) {
	h.eventsMu.Lock()
	set, ok := h.events[ec.userID]
	if !ok {
		set = map[*eventConn]struct{}{}
		h.events[ec.userID] = set
	}
	set[ec] = struct{}{}
	h.eventsMu.Unlock()
	h.metrics.MarkEventConnected(ec.userID)
}

func (h *RealtimeHandler) unregisterEventConn(ec *eventConn) {
	h.eventsMu.Lock()
	if set := h.events[ec.userID]; set != nil {
		delete(set, ec)
		if len(set) == 0 {
			delete(h.events, ec.userID)
		}
	}
	h.eventsMu.Unlock()
	h.metrics.MarkEventDisconnected(ec.userID)
	ec.close()
}

func (ec *eventConn) send(w eventWire) {
	select {
	case ec.out <- w:
	default:
		// Drop on overflow; client will have to reconnect + resume. This
		// is preferable to blocking the bus fanout goroutine.
		log.Printf("[realtime/events] send queue overflow user=%d", ec.userID)
	}
}

func (ec *eventConn) writer() {
	ping := time.NewTicker(wsPingInterval)
	defer ping.Stop()
	for {
		select {
		case <-ec.done:
			return
		case w := <-ec.out:
			ec.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := ec.conn.WriteJSON(w); err != nil {
				ec.close()
				return
			}
		case <-ping.C:
			ec.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := ec.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				ec.close()
				return
			}
		}
	}
}

func (ec *eventConn) reader() {
	defer ec.close()
	for {
		_, raw, err := ec.conn.ReadMessage()
		if err != nil {
			return
		}
		// Handle post-auth resume frames: clients reconnecting after a
		// network blip may send {type:"resume", since_rev:N} without
		// having embedded the hint in the original auth frame (§2.8).
		var frame authFrame
		if json.Unmarshal(raw, &frame) != nil {
			continue
		}
		if frame.Type != "resume" {
			continue // ignore anything else to keep the surface small
		}
		r, err := ec.h.bus.EventsSinceRev(context.Background(), ec.userID, frame.SinceRev)
		if err != nil {
			continue
		}
		if r.Truncated {
			// Client drifted past the retained window → tell it to
			// rebuild from a fresh snapshot.
			ec.send(eventWire{Type: "snapshot_required"})
			continue
		}
		for _, e := range r.Events {
			ec.send(eventWire{
				ID:        e.ID,
				Type:      e.Type,
				Ts:        e.Ts,
				ServerRev: e.ServerRev,
				Data:      e.Data,
			})
		}
	}
}

func (ec *eventConn) close() {
	ec.once.Do(func() {
		close(ec.done)
		_ = ec.conn.Close()
	})
}

// buildSnapshot gathers the user's current device list + favorites so a
// new events-WS client starts with a consistent local cache (搂2.8).
func (h *RealtimeHandler) buildSnapshot(ctx context.Context, userID uint) (map[string]interface{}, error) {
	devices, err := h.devices.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	favorites, err := h.favorites.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Enrich devices with derived online/logged_in.
	deviceIDs := make([]string, 0, len(devices))
	for _, d := range devices {
		deviceIDs = append(deviceIDs, d.DeviceID)
	}
	online := h.presence.BulkOnline(ctx, deviceIDs)
	remarks := map[string]string{}
	var uds []models.UserDevice
	h.db.WithContext(ctx).
		Where("user_id = ? AND device_id IN ?", userID, deviceIDs).
		Find(&uds)
	for _, u := range uds {
		remarks[u.DeviceID] = u.Remark
	}
	deviceItems := make([]map[string]interface{}, 0, len(devices))
	for i := range devices {
		d := &devices[i]
		lastSeen := ""
		if !d.LastSeenAt.IsZero() {
			lastSeen = d.LastSeenAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		deviceItems = append(deviceItems, map[string]interface{}{
			"device_id":    d.DeviceID,
			"device_name": d.DeviceName,
			"remark":       remarks[d.DeviceID],
			"online":       online[d.DeviceID],
			"logged_in":    d.LoggedIn && online[d.DeviceID],
			"access_code":  d.AccessCode,
			"os":           d.OS,
			"os_version":   d.OSVersion,
			"app_version":  d.AppVersion,
			"last_seen_at": lastSeen,
		})
	}
	favItems := make([]map[string]interface{}, 0, len(favorites))
	for _, f := range favorites {
		favItems = append(favItems, map[string]interface{}{
			"device_id":       f.DeviceID,
			"device_name":     f.DeviceName,
			"access_password": f.AccessPassword,
		})
	}
	return map[string]interface{}{
		"devices":   deviceItems,
		"favorites": favItems,
	}, nil
}

// =====================================================================
// /v1/realtime/signal
// =====================================================================

// signalConn is a single signaling WS. role + device_id + (client_id for
// clients) are immutable after auth_ok.
type signalConn struct {
	h        *RealtimeHandler
	conn     *websocket.Conn
	role     service.SignalRole
	deviceID string
	clientID string // empty for host role
	out      chan []byte
	done     chan struct{}
	once     sync.Once
}

// HandleSignal serves GET /v1/realtime/signal.
func (h *RealtimeHandler) HandleSignal(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	conn.SetReadDeadline(time.Now().Add(firstFrameTimeout))
	var af authFrame
	if err := conn.ReadJSON(&af); err != nil {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(4401, "auth timeout"), time.Now().Add(time.Second))
		_ = conn.Close()
		return
	}
	if af.Type != "auth" || af.SignalToken == "" || af.DeviceID == "" {
		writeSignalError(conn, "AUTH_INVALID", "first frame must supply signal_token + device_id")
		_ = conn.Close()
		return
	}

	// Host tokens are consumed (one-time) since host always mints fresh.
	// Client tokens are validated-and-extended so the Chromium client can
	// reconnect after a brief network disruption without needing Qt to
	// re-verify the access code.
	var payload service.SignalTokenPayload
	if af.Role == "host" {
		payload, err = h.tokens.ConsumeSignalToken(c.Request.Context(), af.SignalToken)
	} else {
		payload, err = h.tokens.ValidateAndExtendSignalToken(c.Request.Context(), af.SignalToken)
	}
	if err != nil {
		writeSignalError(conn, "AUTH_INVALID", "signal_token invalid or expired")
		_ = conn.Close()
		return
	}
	if payload.DeviceID != af.DeviceID {
		writeSignalError(conn, "AUTH_INVALID", "device_id mismatch")
		_ = conn.Close()
		return
	}
	if string(payload.Role) != af.Role {
		writeSignalError(conn, "AUTH_INVALID", "role mismatch")
		_ = conn.Close()
		return
	}

	// Before we register the WS as a presence signal for a host, make
	// sure the device row exists. For clients we verify the target host
	// is currently online 鈥?otherwise return a clean HOST_OFFLINE
	// (scenario 37).
	if payload.Role == service.SignalRoleClient {
		if !h.presence.IsOnline(c.Request.Context(), payload.DeviceID) {
			writeSignalError(conn, "HOST_OFFLINE", "Host is offline")
			_ = conn.Close()
			return
		}
	}

	// Tell the client the handshake is complete; assign a server session
	// id so it shows up in logs and errors.
	sessionID := "sig_" + uuid.NewString()
	conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
		return nil
	})

	sc := &signalConn{
		h:        h,
		conn:     conn,
		role:     payload.Role,
		deviceID: payload.DeviceID,
		// R7: prefer the client_id declared by the caller in the auth
		// frame (matches §2.26 "client_id 在首帧 auth 时声明"). Fall
		// back to whatever the token was minted with (Qt today doesn't
		// forward client_id to verify, so payload.ClientID is usually
		// empty), and as a last resort assign a server-side UUID so we
		// never have "" as a map key. For host role the field is
		// ignored anyway.
		clientID: firstNonEmpty(af.ClientID, payload.ClientID),
		out:      make(chan []byte, 64),
		done:     make(chan struct{}),
	}
	if err := conn.WriteJSON(map[string]interface{}{
		"type":       "auth_ok",
		"session_id": sessionID,
	}); err != nil {
		_ = conn.Close()
		return
	}

	// Register in topology. For hosts we also record presence so the
	// online derivation (搂2.4) flips immediately.
	if err := h.registerSignalConn(sc); err != nil {
		writeSignalError(conn, "CONFLICT", err.Error())
		_ = conn.Close()
		return
	}
	defer h.unregisterSignalConn(sc)

	if sc.role == service.SignalRoleHost {
		if err := h.presence.MarkWSConnected(c.Request.Context(), sc.deviceID); err == nil {
			if h.presence.IsOnline(c.Request.Context(), sc.deviceID) && h.presence.RememberOnlineCandidate(c.Request.Context(), sc.deviceID) {
				if d, err := h.devices.GetByDeviceID(c.Request.Context(), sc.deviceID); err == nil && d.UserID != nil {
					h.bus.Publish(c.Request.Context(), service.Event{
						Type:     service.EventDeviceOnlineChanged,
						UserID:   *d.UserID,
						DeviceID: sc.deviceID,
						Data: map[string]interface{}{
							"device_id": sc.deviceID,
							"online":    true,
							"logged_in": d.LoggedIn,
						},
					})
				}
			}
		}
	}

	go sc.writer()
	sc.reader()
}

// registerSignalConn places sc into the topology maps. For host role,
// latest connection wins: the existing host WS is closed and replaced,
// so a reconnecting host process always succeeds.
func (h *RealtimeHandler) registerSignalConn(sc *signalConn) error {
	h.signalMu.Lock()
	defer h.signalMu.Unlock()
	switch sc.role {
	case service.SignalRoleHost:
		if _, exists := h.signalHosts[sc.deviceID]; exists {
			// Keep latest wins: close the old one and replace. This
			// matches the operator expectation "reconnect should work".
			old := h.signalHosts[sc.deviceID]
			h.signalHosts[sc.deviceID] = sc
			go old.close()
		} else {
			h.signalHosts[sc.deviceID] = sc
		}
		h.metrics.MarkSignalConnected(sc.role, sc.deviceID, sc.clientID)
	case service.SignalRoleClient:
		if sc.clientID == "" {
			sc.clientID = "cli_" + uuid.NewString()
		}
		clients, ok := h.signalClient[sc.deviceID]
		if !ok {
			clients = map[string]*signalConn{}
			h.signalClient[sc.deviceID] = clients
		}
		clients[sc.clientID] = sc
		h.metrics.MarkSignalConnected(sc.role, sc.deviceID, sc.clientID)
	default:
		return fmt.Errorf("unknown role %q", sc.role)
	}
	return nil
}

func (h *RealtimeHandler) unregisterSignalConn(sc *signalConn) {
	h.signalMu.Lock()
	switch sc.role {
	case service.SignalRoleHost:
		if cur := h.signalHosts[sc.deviceID]; cur == sc {
			delete(h.signalHosts, sc.deviceID)
			h.metrics.MarkSignalDisconnected(sc.role, sc.deviceID, sc.clientID)
			// Mark presence down for this instance.
			_ = h.presence.MarkWSDisconnected(context.Background(), sc.deviceID)
			// Notify any clients still connected to this host.
			for _, cc := range h.signalClient[sc.deviceID] {
				cc.send(mustMarshal(map[string]interface{}{
					"type": "error",
					"code": "PEER_DISCONNECTED",
				}))
				go cc.close()
			}
			// Publish online-changed once if we were the last live signal.
			if online := h.presence.IsOnline(context.Background(), sc.deviceID); !online && h.presence.ForgetOnlineCandidate(context.Background(), sc.deviceID) {
				if d, err := h.devices.GetByDeviceID(context.Background(), sc.deviceID); err == nil && d.UserID != nil {
					h.bus.Publish(context.Background(), service.Event{
						Type:     service.EventDeviceOnlineChanged,
						UserID:   *d.UserID,
						DeviceID: sc.deviceID,
						Data: map[string]interface{}{
							"device_id": sc.deviceID,
							"online":    false,
							"logged_in": false,
						},
					})
				}
			}
		}
	case service.SignalRoleClient:
		if m := h.signalClient[sc.deviceID]; m != nil {
			if cur := m[sc.clientID]; cur == sc {
				delete(m, sc.clientID)
				h.metrics.MarkSignalDisconnected(sc.role, sc.deviceID, sc.clientID)
				if len(m) == 0 {
					delete(h.signalClient, sc.deviceID)
				}
			}
		}
	}
	h.signalMu.Unlock()
	sc.close()
}

// reader shuttles incoming signaling frames (SDP/ICE/jingle) between host
// and its clients. Per 搂2.26, jingle frames carry `client_id` so we can
// route the host's reply to the right client.
func (sc *signalConn) reader() {
	defer sc.close()
	for {
		_, msg, err := sc.conn.ReadMessage()
		if err != nil {
			return
		}
		sc.h.routeSignalMessage(sc, msg)
	}
}

func (sc *signalConn) writer() {
	ping := time.NewTicker(wsPingInterval)
	defer ping.Stop()
	for {
		select {
		case <-sc.done:
			return
		case msg := <-sc.out:
			sc.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := sc.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				sc.close()
				return
			}
		case <-ping.C:
			sc.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := sc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				sc.close()
				return
			}
		}
	}
}

func (sc *signalConn) send(msg []byte) {
	select {
	case sc.out <- msg:
	default:
		log.Printf("[realtime/signal] send queue overflow device=%s role=%s", sc.deviceID, sc.role)
	}
}

func (sc *signalConn) close() {
	sc.once.Do(func() {
		close(sc.done)
		_ = sc.conn.Close()
	})
}

// routeSignalMessage forwards a frame based on direction + client_id.
//   host 鈫?server: expect {client_id:...} in payload, route to that client.
//   client 鈫?server: always route to the host for this device.
func (h *RealtimeHandler) routeSignalMessage(from *signalConn, raw []byte) {
	switch from.role {
	case service.SignalRoleHost:
		// Peek at client_id without rewriting the payload.
		var peek struct {
			ClientID string `json:"client_id"`
		}
		_ = json.Unmarshal(raw, &peek)
		h.signalMu.RLock()
		clients := h.signalClient[from.deviceID]
		target := clients[peek.ClientID]
		h.signalMu.RUnlock()
		if target == nil {
			return
		}
		target.send(raw)
	case service.SignalRoleClient:
		h.signalMu.RLock()
		host := h.signalHosts[from.deviceID]
		h.signalMu.RUnlock()
		if host == nil {
			from.send(mustMarshal(map[string]interface{}{
				"type": "error",
				"code": "PEER_DISCONNECTED",
			}))
			go from.close()
			return
		}
		// Inject client_id so the host can address replies back.
		injected, err := injectClientID(raw, from.clientID)
		if err != nil {
			return
		}
		host.send(injected)
	}
}

// writeSignalError writes a single JSON error frame to the connection.
func writeSignalError(conn *websocket.Conn, code, detail string) {
	conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	_ = conn.WriteJSON(map[string]interface{}{
		"type":   "error",
		"code":   code,
		"detail": detail,
	})
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// firstNonEmpty returns the first non-empty string from its arguments,
// or "" if all are empty. Used by HandleSignal to pick the best
// client_id candidate (auth frame > token payload).
func firstNonEmpty(candidates ...string) string {
	for _, s := range candidates {
		if s != "" {
			return s
		}
	}
	return ""
}

// injectClientID parses the JSON payload, ensures a client_id field exists
// (overwriting any supplied value), and re-serialises. Keeps the wire
// shape the host sees consistent even if the client forgets to set it.
func injectClientID(raw []byte, clientID string) ([]byte, error) {
	if !strings.HasPrefix(strings.TrimSpace(string(raw)), "{") {
		return raw, nil // not an object, pass through
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw, err
	}
	obj["client_id"] = clientID
	return json.Marshal(obj)
}
