package service

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// metricRouteKey is the label set of one per-route request counter.
type metricRouteKey struct {
	Route  string
	Method string
	Status int
}

// HTTPRouteRequests is one labelled request counter: how many requests matched
// `Route` with that method and ended in that status code.
type HTTPRouteRequests struct {
	Route  string `json:"route"`
	Method string `json:"method"`
	Status int    `json:"status"`
	Count  int64  `json:"count"`
}

type MetricsService struct {
	apiRequestsTotal atomic.Int64
	apiRequestsToday atomic.Int64
	apiDay           atomic.Int64

	mu            sync.RWMutex
	signalHosts   map[string]struct{}
	signalClients map[string]map[string]struct{}
	eventUsers    map[uint]int
	httpByRoute   map[metricRouteKey]int64
}

type MetricsSnapshot struct {
	APIRequestsTotal     int64 `json:"api_requests_total"`
	APIRequestsToday     int64 `json:"api_requests_today"`
	WebSocketConnections int   `json:"websocket_connections"`
	SignalHosts          int   `json:"signal_hosts"`
	SignalClients        int   `json:"signal_clients"`
	EventStreams         int   `json:"event_streams"`
	// HTTPRouteRequests is ordered by route, method and status so the
	// exposition is byte-stable between scrapes of an idle process.
	HTTPRouteRequests []HTTPRouteRequests `json:"http_route_requests"`
}

func NewMetricsService() *MetricsService {
	m := &MetricsService{
		signalHosts:   map[string]struct{}{},
		signalClients: map[string]map[string]struct{}{},
		eventUsers:    map[uint]int{},
		httpByRoute:   map[metricRouteKey]int64{},
	}
	m.apiDay.Store(dayKey(time.Now().UTC()))
	return m
}

// RecordHTTPRequest counts one finished request under /v1.
//
// `path` is the concrete request path and decides whether the request counts at
// all (the endpoint is about the API, not about /health or /metrics). `route`
// is gin's matched pattern and is what the labels use: labelling on the raw path
// would give every device id and recording id its own time series.
func (m *MetricsService) RecordHTTPRequest(method, path, route string, statusCode int) {
	if m == nil || !strings.HasPrefix(path, "/v1/") {
		return
	}
	m.rollToday(time.Now().UTC())
	m.apiRequestsTotal.Add(1)
	m.apiRequestsToday.Add(1)

	// A request that matched no route is counted as "unmatched" rather than
	// dropped: a 404 storm is a signal, and dropping it would hide exactly the
	// traffic an operator is looking for.
	label := strings.TrimSpace(route)
	if label == "" {
		label = "unmatched"
	}
	key := metricRouteKey{
		Route:  label,
		Method: strings.ToUpper(strings.TrimSpace(method)),
		Status: statusCode,
	}
	m.mu.Lock()
	m.httpByRoute[key]++
	m.mu.Unlock()
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
	byRoute := make([]HTTPRouteRequests, 0, len(m.httpByRoute))
	for key, count := range m.httpByRoute {
		byRoute = append(byRoute, HTTPRouteRequests{
			Route: key.Route, Method: key.Method, Status: key.Status, Count: count,
		})
	}
	sort.Slice(byRoute, func(i, j int) bool {
		if byRoute[i].Route != byRoute[j].Route {
			return byRoute[i].Route < byRoute[j].Route
		}
		if byRoute[i].Method != byRoute[j].Method {
			return byRoute[i].Method < byRoute[j].Method
		}
		return byRoute[i].Status < byRoute[j].Status
	})

	return MetricsSnapshot{
		APIRequestsTotal:     m.apiRequestsTotal.Load(),
		APIRequestsToday:     m.apiRequestsToday.Load(),
		WebSocketConnections: len(m.signalHosts) + signalClients + eventStreams,
		SignalHosts:          len(m.signalHosts),
		SignalClients:        signalClients,
		EventStreams:         eventStreams,
		HTTPRouteRequests:    byRoute,
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
