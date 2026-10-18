package service

import "testing"

func TestMetricsSnapshotCountsRequestsAndConnections(t *testing.T) {
	metrics := NewMetricsService()
	metrics.RecordHTTPRequest("GET", "/v1/devices", "/v1/me/devices", 200)
	metrics.RecordHTTPRequest("GET", "/health", "", 200) // not an API request
	metrics.MarkSignalConnected(SignalRoleHost, "device-1", "")
	metrics.MarkSignalConnected(SignalRoleClient, "device-1", "client-1")
	metrics.MarkEventConnected(7)

	snapshot := metrics.Snapshot()
	if snapshot.APIRequestsTotal != 1 || snapshot.APIRequestsToday != 1 {
		t.Fatalf("request counters = total %d, today %d; want 1, 1",
			snapshot.APIRequestsTotal, snapshot.APIRequestsToday)
	}
	if snapshot.SignalHosts != 1 || snapshot.SignalClients != 1 || snapshot.EventStreams != 1 {
		t.Fatalf("connections = hosts %d, clients %d, events %d; want 1, 1, 1",
			snapshot.SignalHosts, snapshot.SignalClients, snapshot.EventStreams)
	}
	if snapshot.WebSocketConnections != 3 {
		t.Fatalf("websocket connections = %d, want 3", snapshot.WebSocketConnections)
	}

	metrics.MarkSignalDisconnected(SignalRoleHost, "device-1", "")
	metrics.MarkSignalDisconnected(SignalRoleClient, "device-1", "client-1")
	metrics.MarkEventDisconnected(7)
	snapshot = metrics.Snapshot()
	if snapshot.WebSocketConnections != 0 {
		t.Fatalf("websocket connections after disconnect = %d, want 0",
			snapshot.WebSocketConnections)
	}
	if snapshot.APIRequestsTotal != 1 {
		t.Fatalf("total request counter reset with a live gauge: %d", snapshot.APIRequestsTotal)
	}
}

// The labels are bounded by design: many device ids share one route pattern,
// and a request that matched nothing lands in "unmatched" rather than creating a
// series per bogus path.
func TestMetricsLabelsRequestsByRouteNotPath(t *testing.T) {
	metrics := NewMetricsService()
	metrics.RecordHTTPRequest("get", "/v1/me/devices/111/heartbeat", "/v1/me/devices/:device_id/heartbeat", 200)
	metrics.RecordHTTPRequest("GET", "/v1/me/devices/222/heartbeat", "/v1/me/devices/:device_id/heartbeat", 200)
	metrics.RecordHTTPRequest("POST", "/v1/me/devices/111/heartbeat", "/v1/me/devices/:device_id/heartbeat", 403)
	metrics.RecordHTTPRequest("GET", "/v1/nope/1234", "", 404)

	got := metrics.Snapshot().HTTPRouteRequests
	want := []HTTPRouteRequests{
		{Route: "/v1/me/devices/:device_id/heartbeat", Method: "GET", Status: 200, Count: 2},
		{Route: "/v1/me/devices/:device_id/heartbeat", Method: "POST", Status: 403, Count: 1},
		{Route: "unmatched", Method: "GET", Status: 404, Count: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("series = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("series[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	// Ordering is part of the contract: a scrape of an idle process must not
	// shuffle lines, or every comparison of two scrapes shows a diff.
	first := metrics.Snapshot().HTTPRouteRequests
	for i := 0; i < 5; i++ {
		next := metrics.Snapshot().HTTPRouteRequests
		for j := range next {
			if next[j] != first[j] {
				t.Fatalf("snapshot order changed: %+v then %+v", first, next)
			}
		}
	}
}
