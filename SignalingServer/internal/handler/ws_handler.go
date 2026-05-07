package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	pingPeriod = 60 * time.Second
	pongWait   = 90 * time.Second
)

// WebSocket message types
type WSMessage struct {
	Type     string `json:"type"`
	Password string `json:"password,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// ConnectionInfo stores information about a WebSocket connection
type ConnectionInfo struct {
	Conn     *websocket.Conn
	DeviceID string
	Role     string // "host" or "client"
	ClientID string // Only for clients, unique identifier
}

// jingleIQ is a minimal struct for extracting session ID from Jingle XML
type jingleIQ struct {
	XMLName xml.Name   `xml:"iq"`
	Jingle  jingleElem `xml:"jingle"`
}

type jingleElem struct {
	SID string `xml:"sid,attr"`
}

// extractSessionID extracts the Jingle session ID (sid) from a Jingle XML message.
// Returns empty string if the message is not a Jingle stanza or has no sid.
func extractSessionID(message []byte) string {
	var iq jingleIQ
	if err := xml.Unmarshal(message, &iq); err != nil {
		return ""
	}
	return iq.Jingle.SID
}

type WSHandler struct {
	connections    map[string]*ConnectionInfo // key: connection key
	deviceClients  map[string][]string        // key: deviceID, value: []clientID
	sessionClients map[string]string           // key: "deviceID:sid", value: clientID
	userSyncConns  map[uint][]*websocket.Conn  // key: userID, value: list of sync WebSocket connections
	mu             sync.RWMutex
	deviceService  *service.DeviceService
	authService    *service.AuthService
	apiKeyAuth     *middleware.APIKeyAuth
	db             *gorm.DB
	rdb            *redis.Client
}

func NewWSHandler(deviceService *service.DeviceService, authService *service.AuthService, db *gorm.DB, rdb *redis.Client) *WSHandler {
	return &WSHandler{
		connections:    make(map[string]*ConnectionInfo),
		deviceClients:  make(map[string][]string),
		sessionClients: make(map[string]string),
		userSyncConns:  make(map[uint][]*websocket.Conn),
		deviceService:  deviceService,
		authService:    authService,
		db:             db,
		rdb:            rdb,
	}
}

func (h *WSHandler) SetAPIKeyAuth(auth *middleware.APIKeyAuth) {
	h.apiKeyAuth = auth
}

// generateRandomHex generates a random hex string
func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// HandleWebSocket handles WebSocket connections
// Route: /signal/:device_id?access_code=xxx (Host)
// Route: /client/:device_id/:access_code (Client)
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	if h.apiKeyAuth != nil && !h.apiKeyAuth.ValidateRequest(c) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "INVALID_API_KEY",
			"message": "Invalid or missing API key",
		})
		return
	}

	deviceID := c.Param("device_id")
	accessCode := c.Query("access_code")
	
	// Check if it's a client connection (path contains /client/)
	isClient := c.FullPath() == "/client/:device_id/:access_code"
	if isClient {
		// For client, access_code is in path parameter
		accessCode = c.Param("access_code")
	}
	
	log.Printf("WebSocket connection request: device_id=%s, isClient=%v", deviceID, isClient)
	
	// Host connection: Auto-register if not exists, create temp password
	if !isClient {
		// Check if device exists
		_, err := h.deviceService.GetByDeviceID(c.Request.Context(), deviceID)
		if err != nil {
			// Device doesn't exist, register it
			log.Printf("Device %s not registered, auto-registering...", deviceID)
			req := &service.RegisterDeviceRequest{
				OS:         "Unknown",
				OSVersion:  "Unknown",
				AppVersion: "1.0.0",
			}
			
			// Use provided device_id
			device, err := h.deviceService.RegisterDeviceWithID(c.Request.Context(), deviceID, req)
			if err != nil {
				log.Printf("Failed to register device: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register device"})
				return
			}
			
			// Generate and set temporary password
			tempPassword := h.authService.GenerateTemporaryPassword()
			if accessCode == "" {
				accessCode = tempPassword
			}
			err = h.authService.SetTemporaryPassword(c.Request.Context(), deviceID, accessCode)
			if err != nil {
				log.Printf("Failed to set temporary password: %v", err)
			}
			
			log.Printf("Device registered: device_id=%s, temp_password=%s", device.DeviceID, accessCode)
		} else {
			// Device exists, update temporary password if provided
			if accessCode != "" {
				err = h.authService.SetTemporaryPassword(c.Request.Context(), deviceID, accessCode)
				if err != nil {
					log.Printf("Failed to update temporary password: %v", err)
				} else {
					log.Printf("Updated temporary password for device_id=%s", deviceID)
				}
			}
		}
	} else {
		// Client connection: Verify access code
		if !h.authService.VerifyDevice(c.Request.Context(), deviceID, accessCode) {
			log.Printf("Authentication failed for client: device_id=%s", deviceID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
	}
	
	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()
	
	var role string
	var connectionKey string
	var clientID string
	
	if isClient {
		role = "client"
		// Generate unique client ID
		clientID = fmt.Sprintf("%s_%s", deviceID, generateRandomHex(4))
		connectionKey = fmt.Sprintf("%s_client_%s", deviceID, clientID)
		log.Printf("Client %s attempting to connect to device %s", clientID, deviceID)
	} else {
		role = "host"
		connectionKey = fmt.Sprintf("%s_host", deviceID)
		log.Printf("Host connecting for device %s", deviceID)
	}
	
	// Create connection info
	connInfo := &ConnectionInfo{
		Conn:     conn,
		DeviceID: deviceID,
		Role:     role,
		ClientID: clientID,
	}
	
	// Register connection
	h.registerConnection(connectionKey, connInfo, isClient)
	defer h.unregisterConnection(connectionKey, deviceID, clientID, isClient)
	
	// Set device online (only for host)
	if !isClient {
		h.deviceService.SetDeviceOnline(context.Background(), deviceID, true)
		h.NotifyDeviceOnlineStatus(deviceID, true)
		defer func() {
			h.deviceService.SetDeviceOnline(context.Background(), deviceID, false)
			// Clear logged_in when device disconnects from signaling
			h.db.Model(&models.Device{}).Where("device_id = ?", deviceID).Update("logged_in", false)
			h.NotifyDeviceOnlineStatus(deviceID, false)
		}()
	}
	
	log.Printf("WebSocket connected: device_id=%s, role=%s, connection_key=%s", deviceID, role, connectionKey)
	if isClient {
		h.mu.RLock()
		clientCount := len(h.deviceClients[deviceID])
		h.mu.RUnlock()
		log.Printf("Client %s connected successfully. Total clients for device %s: %d", clientID, deviceID, clientCount)
	}

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Message handling loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for %s (%s): %v", connectionKey, role, err)
			} else {
				log.Printf("WebSocket closed for %s (%s)", connectionKey, role)
			}
			break
		}
		
		log.Printf("Received message from %s (%s): %d bytes", connectionKey, role, len(message))
		
		// Try to parse as JSON to check for special message types
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err == nil {
			// Handle special Host -> Server messages
			if !isClient && wsMsg.Type == "set_temp_password" {
				// Host is setting/updating temporary password
				if wsMsg.Password != "" {
					err := h.authService.SetTemporaryPassword(context.Background(), deviceID, wsMsg.Password)
					if err != nil {
						log.Printf("Failed to set temp password for device %s: %v", deviceID, err)
						h.sendToConnection(conn, map[string]interface{}{
							"type":    "error",
							"message": "Failed to set password",
						})
					} else {
						log.Printf("Temporary password set for device %s", deviceID)
						h.sendToConnection(conn, map[string]interface{}{
							"type": "password_set",
						})
					}
				}
				continue // Don't forward this message to clients
			}
		}
		
		// Forward message
		if isClient {
			// Client -> Host (also registers session-to-client mapping)
			h.forwardToHost(deviceID, clientID, message)
		} else {
			// Host -> Client (route by session ID, fallback to broadcast)
			h.routeToClient(deviceID, message)
		}
	}
}

// sendToConnection sends a JSON message to a specific WebSocket connection
func (h *WSHandler) sendToConnection(conn *websocket.Conn, msg map[string]interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// registerConnection registers a WebSocket connection
func (h *WSHandler) registerConnection(connectionKey string, connInfo *ConnectionInfo, isClient bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Close existing connection if any
	if existingConn, exists := h.connections[connectionKey]; exists {
		log.Printf("Closing existing connection for connection_key=%s", connectionKey)
		existingConn.Conn.Close()
	}
	
	h.connections[connectionKey] = connInfo
	
	// Track client for this device
	if isClient {
		h.deviceClients[connInfo.DeviceID] = append(h.deviceClients[connInfo.DeviceID], connInfo.ClientID)
	}
	
	log.Printf("Connection registered: connection_key=%s, role=%s (total connections: %d)", 
		connectionKey, connInfo.Role, len(h.connections))
}

// unregisterConnection unregisters a WebSocket connection
func (h *WSHandler) unregisterConnection(connectionKey string, deviceID string, clientID string, isClient bool) {
	h.mu.Lock()
	delete(h.connections, connectionKey)
	
	// Remove client from device's client list
	if isClient && clientID != "" {
		clients := h.deviceClients[deviceID]
		for i, cid := range clients {
			if cid == clientID {
				h.deviceClients[deviceID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		
		// Clean up session mappings for this client
		for key, cid := range h.sessionClients {
			if cid == clientID {
				delete(h.sessionClients, key)
				log.Printf("Session mapping removed: %s (client %s disconnected)", key, clientID)
			}
		}
		
		// Clean up empty client list
		if len(h.deviceClients[deviceID]) == 0 {
			delete(h.deviceClients, deviceID)
		}
		
		log.Printf("Client %s removed from device %s. Remaining clients: %d", 
			clientID, deviceID, len(h.deviceClients[deviceID]))
	}
	
	remainingConnections := len(h.connections)
	h.mu.Unlock()
	
	// For Host disconnection, clean up Redis temporary password
	if !isClient {
		log.Printf("Host disconnected, cleaning up temporary password for device %s", deviceID)
		if err := h.authService.ClearTemporaryPassword(context.Background(), deviceID); err != nil {
			log.Printf("Failed to clear temporary password for device %s: %v", deviceID, err)
		}
	}
	
	log.Printf("Connection unregistered: connection_key=%s (remaining connections: %d)", 
		connectionKey, remainingConnections)
}

// broadcastToClients broadcasts a message from Host to all connected Clients
func (h *WSHandler) broadcastToClients(deviceID string, message []byte) {
	h.mu.RLock()
	clientIDs := make([]string, len(h.deviceClients[deviceID]))
	copy(clientIDs, h.deviceClients[deviceID])
	h.mu.RUnlock()
	
	if len(clientIDs) == 0 {
		log.Printf("No clients connected to device %s, message not forwarded", deviceID)
		return
	}
	
	log.Printf("Broadcasting message from Host %s to %d client(s)", deviceID, len(clientIDs))
	
	disconnectedClients := []string{}
	
	for _, clientID := range clientIDs {
		connectionKey := fmt.Sprintf("%s_client_%s", deviceID, clientID)
		
		h.mu.RLock()
		connInfo, exists := h.connections[connectionKey]
		h.mu.RUnlock()
		
		if !exists {
			log.Printf("Client %s not found in connections", clientID)
			disconnectedClients = append(disconnectedClients, clientID)
			continue
		}
		
		err := connInfo.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Failed to send to client %s: %v", clientID, err)
			disconnectedClients = append(disconnectedClients, clientID)
		} else {
			log.Printf("Server -> Client %s: Forwarded %d bytes", clientID, len(message))
		}
	}
	
	// Clean up disconnected clients
	if len(disconnectedClients) > 0 {
		h.mu.Lock()
		for _, clientID := range disconnectedClients {
			clients := h.deviceClients[deviceID]
			for i, cid := range clients {
				if cid == clientID {
					h.deviceClients[deviceID] = append(clients[:i], clients[i+1:]...)
					break
				}
			}
		}
		h.mu.Unlock()
		log.Printf("Removed %d disconnected client(s) from device %s", len(disconnectedClients), deviceID)
	}
}

// routeToClient routes a Host message to the correct Client by session ID.
// If the message contains a Jingle sid that maps to a known client, it is sent
// only to that client. Otherwise it falls back to broadcasting to all clients.
func (h *WSHandler) routeToClient(deviceID string, message []byte) {
	sid := extractSessionID(message)
	if sid != "" {
		sessionKey := fmt.Sprintf("%s:%s", deviceID, sid)

		h.mu.RLock()
		targetClientID, found := h.sessionClients[sessionKey]
		h.mu.RUnlock()

		if found {
			connectionKey := fmt.Sprintf("%s_client_%s", deviceID, targetClientID)

			h.mu.RLock()
			connInfo, exists := h.connections[connectionKey]
			h.mu.RUnlock()

			if exists {
				err := connInfo.Conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Failed to send to client %s (session %s): %v", targetClientID, sid, err)
				} else {
					log.Printf("Server -> Client %s: Routed %d bytes (session %s)", targetClientID, len(message), sid)
				}
				return
			}
			log.Printf("Client %s for session %s not found in connections, falling back to broadcast", targetClientID, sid)
		}
	}

	// Fallback: broadcast to all clients (non-Jingle messages or unknown session)
	h.broadcastToClients(deviceID, message)
}

// forwardToHost forwards a message from Client to Host
func (h *WSHandler) forwardToHost(deviceID string, clientID string, message []byte) {
	// Register session-to-client mapping from Jingle messages
	if sid := extractSessionID(message); sid != "" {
		sessionKey := fmt.Sprintf("%s:%s", deviceID, sid)
		h.mu.Lock()
		if existing, ok := h.sessionClients[sessionKey]; !ok || existing != clientID {
			h.sessionClients[sessionKey] = clientID
			log.Printf("Session mapping registered: sid=%s -> client=%s (device=%s)", sid, clientID, deviceID)
		}
		h.mu.Unlock()
	}

	hostKey := fmt.Sprintf("%s_host", deviceID)
	
	h.mu.RLock()
	hostConn, exists := h.connections[hostKey]
	h.mu.RUnlock()
	
	if !exists {
		log.Printf("Host not connected for device %s (from client %s)", deviceID, clientID)
		return
	}
	
	err := hostConn.Conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		log.Printf("Failed to forward message to host %s: %v", deviceID, err)
	} else {
		log.Printf("Server -> Host: Forwarded %d bytes from client %s", len(message), clientID)
	}
}

// IsHostOnline checks if a host is online (has active WebSocket connection)
func (h *WSHandler) IsHostOnline(deviceID string) bool {
	hostKey := fmt.Sprintf("%s_host", deviceID)
	
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	_, exists := h.connections[hostKey]
	return exists
}
// GetConnectionCount returns the total number of active WebSocket connections
func (h *WSHandler) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.connections)
}

// HandleUserSync handles GET /api/v1/user/sync?token=xxx
// User-level WebSocket for real-time sync notifications.
func (h *WSHandler) HandleUserSync(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}

	// Validate token: look up Redis key user_token:{token} to get userID
	val, err := h.rdb.Get(context.Background(), fmt.Sprintf("user_token:%s", token)).Result()
	if err != nil {
		log.Printf("User sync: invalid or expired token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	userID64, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		log.Printf("User sync: invalid userID in Redis: %s", val)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid session"})
		return
	}
	userID := uint(userID64)

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("User sync: failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Register in userSyncConns
	h.mu.Lock()
	h.userSyncConns[userID] = append(h.userSyncConns[userID], conn)
	h.mu.Unlock()

	log.Printf("User sync: connected for userID=%d (total sync conns: %d)", userID, len(h.userSyncConns[userID]))

	// On disconnect, remove from userSyncConns
	defer func() {
		h.mu.Lock()
		conns := h.userSyncConns[userID]
		for i, c := range conns {
			if c == conn {
				h.userSyncConns[userID] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(h.userSyncConns[userID]) == 0 {
			delete(h.userSyncConns, userID)
		}
		h.mu.Unlock()
		log.Printf("User sync: disconnected for userID=%d", userID)
	}()

	// Read loop: keep alive, discard messages, handle pong
	conn.SetPongHandler(func(appData string) error {
		return nil
	})
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("User sync: WebSocket error for userID=%d: %v", userID, err)
			}
			break
		}
	}
}

// NotifyUserSync sends a sync message to all WebSocket connections for a user.
func (h *WSHandler) NotifyUserSync(userID uint, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("NotifyUserSync: failed to marshal message: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	conns := h.userSyncConns[userID]
	if len(conns) == 0 {
		return
	}

	var alive []*websocket.Conn
	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("NotifyUserSync: failed to send to userID=%d: %v", userID, err)
			conn.Close()
		} else {
			alive = append(alive, conn)
		}
	}
	h.userSyncConns[userID] = alive
	if len(alive) == 0 {
		delete(h.userSyncConns, userID)
	}
}

// NotifyDeviceOnlineStatus notifies sync connections when a device goes online/offline.
func (h *WSHandler) NotifyDeviceOnlineStatus(deviceID string, online bool) {
	// Look up the device in DB to get UserID
	var device models.Device
	if err := h.db.Where("device_id = ?", deviceID).First(&device).Error; err != nil {
		return
	}
	if device.UserID == nil || *device.UserID == 0 {
		return
	}

	msgType := "device_offline"
	if online {
		msgType = "device_online"
	}

	h.NotifyUserSync(*device.UserID, map[string]interface{}{
		"type":      msgType,
		"device_id": deviceID,
		"logged_in": device.LoggedIn,
	})
}