package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"quickdesk/signaling/internal/service"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

type WSHandler struct {
	connections   map[string]*ConnectionInfo // key: connection key
	deviceClients map[string][]string        // key: deviceID, value: []clientID
	mu            sync.RWMutex
	deviceService *service.DeviceService
	authService   *service.AuthService
}

func NewWSHandler(deviceService *service.DeviceService, authService *service.AuthService) *WSHandler {
	return &WSHandler{
		connections:   make(map[string]*ConnectionInfo),
		deviceClients: make(map[string][]string),
		deviceService: deviceService,
		authService:   authService,
	}
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
		defer h.deviceService.SetDeviceOnline(context.Background(), deviceID, false)
	}
	
	log.Printf("WebSocket connected: device_id=%s, role=%s, connection_key=%s", deviceID, role, connectionKey)
	if isClient {
		// Log total connected clients
		h.mu.RLock()
		clientCount := len(h.deviceClients[deviceID])
		h.mu.RUnlock()
		log.Printf("Client %s connected successfully. Total clients for device %s: %d", clientID, deviceID, clientCount)
	}
	
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
			// Client -> Host
			h.forwardToHost(deviceID, clientID, message)
		} else {
			// Host -> All Clients (broadcast)
			h.broadcastToClients(deviceID, message)
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

// forwardToHost forwards a message from Client to Host
func (h *WSHandler) forwardToHost(deviceID string, clientID string, message []byte) {
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
