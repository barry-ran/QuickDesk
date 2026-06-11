package service

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type MetricsService struct {
	apiRequestsToday atomic.Int64
	apiDay           atomic.Int64

	mu            sync.RWMutex
	signalHosts   map[string]struct{}
	signalClients map[string]map[string]struct{}
	eventUsers     map[uint]int
}

type MetricsSnapshot struct {
	APIRequestsToday    int64 `json:"api_requests_today"`
	WebSocketConnections int   `json:"websocket_connections"`
	SignalHosts          int   `json:"signal_hosts"`
	SignalClients        int   `json:"signal_clients"`
	EventStreams         int   `json:"event_streams"`
}

func NewMetricsService() *MetricsService {
	m := &MetricsService{
		signalHosts:   map[string]struct{}{},
		signalClients: map[string]map[string]struct{}{},
		eventUsers:     map[uint]int{},
	}
	m.apiDay.Store(dayKey(time.Now().UTC()))
	return m
}

func (m *MetricsService) RecordHTTPRequest(path string) {
	if m == nil || !strings.HasPrefix(path, "/v1/") {
		return
	}
	m.rollToday(time.Now().UTC())
	m.apiRequestsToday.Add(1)
}

func (m *MetricsService) MarkSignalConnected(role SignalRole, deviceID, clientID string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	switch role {
	case SignalRoleHost:
		m.signalHosts[deviceID] = struct{}{}
	case SignalRoleClient:
		clients := m.signalClients[deviceID]
		if clients == nil {
			clients = map[string]struct{}{}
			m.signalClients[deviceID] = clients
		}
		clients[clientID] = struct{}{}
	}
}

func (m *MetricsService) MarkSignalDisconnected(role SignalRole, deviceID, clientID string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	switch role {
	case SignalRoleHost:
		delete(m.signalHosts, deviceID)
	case SignalRoleClient:
		if clients := m.signalClients[deviceID]; clients != nil {
			delete(clients, clientID)
			if len(clients) == 0 {
				delete(m.signalClients, deviceID)
			}
		}
	}
}

func (m *MetricsService) MarkEventConnected(userID uint) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.eventUsers[userID]++
	m.mu.Unlock()
}

func (m *MetricsService) MarkEventDisconnected(userID uint) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if n := m.eventUsers[userID]; n <= 1 {
		delete(m.eventUsers, userID)
	} else {
		m.eventUsers[userID] = n - 1
	}
}

func (m *MetricsService) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}
	m.rollToday(time.Now().UTC())
	m.mu.RLock()
	defer m.mu.RUnlock()

	signalClients := 0
	for _, clients := range m.signalClients {
		signalClients += len(clients)
	}
	eventStreams := 0
	for _, n := range m.eventUsers {
		eventStreams += n
	}

	return MetricsSnapshot{
		APIRequestsToday:    m.apiRequestsToday.Load(),
		WebSocketConnections: len(m.signalHosts) + signalClients + eventStreams,
		SignalHosts:          len(m.signalHosts),
		SignalClients:        signalClients,
		EventStreams:         eventStreams,
	}
}

func (m *MetricsService) rollToday(now time.Time) {
	day := dayKey(now)
	for {
		old := m.apiDay.Load()
		if old == day {
			return
		}
		if m.apiDay.CompareAndSwap(old, day) {
			m.apiRequestsToday.Store(0)
			return
		}
	}
}

func dayKey(t time.Time) int64 {
	y, month, day := t.Date()
	return int64(y*10000 + int(month)*100 + day)
}