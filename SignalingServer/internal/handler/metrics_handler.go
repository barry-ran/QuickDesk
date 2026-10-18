package handler

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// MetricsHandler serves the opt-in Prometheus text endpoint.
//
// The endpoint is deliberately outside /v1 and outside API-key auth: Prometheus
// scrapers are infrastructure, not interactive API clients. It is disabled
// unless METRICS_TOKEN is set, and then requires a bearer token, so enabling
// scraping is an explicit deployment decision.
type MetricsHandler struct {
	metrics      *service.MetricsService
	version      string
	metricsToken string
	startedAt    time.Time
}

func NewMetricsHandler(metrics *service.MetricsService, version, metricsToken string) *MetricsHandler {
	return &MetricsHandler{
		metrics:      metrics,
		version:      version,
		metricsToken: strings.TrimSpace(metricsToken),
		startedAt:    time.Now().UTC(),
	}
}

// Prometheus writes the standard text exposition format. Keep the metric names
// and labels stable: dashboards and alert rules key on them.
func (h *MetricsHandler) Prometheus(c *gin.Context) {
	if !validMetricsToken(c.GetHeader("Authorization"), h.metricsToken) {
		c.Header("WWW-Authenticate", `Bearer realm="metrics"`)
		c.String(http.StatusUnauthorized, "unauthorized\n")
		return
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	snapshot := h.metrics.Snapshot()

	var b strings.Builder
	writeMetric := func(name, help, metricType, value string) {
		fmt.Fprintf(&b, "# HELP %s %s\n", name, help)
		fmt.Fprintf(&b, "# TYPE %s %s\n", name, metricType)
		fmt.Fprintf(&b, "%s %s\n", name, value)
	}

	fmt.Fprintf(&b, "# HELP quickdesk_signaling_build_info Build information for the running signaling server.\n")
	fmt.Fprintf(&b, "# TYPE quickdesk_signaling_build_info gauge\n")
	fmt.Fprintf(&b, "quickdesk_signaling_build_info{version=%s,go_version=%s} 1\n",
		strconv.Quote(h.version), strconv.Quote(runtime.Version()))
	writeMetric("quickdesk_signaling_uptime_seconds",
		"Seconds since the signaling process started.", "gauge",
		strconv.FormatFloat(time.Since(h.startedAt).Seconds(), 'f', 3, 64))
	writeMetric("quickdesk_signaling_api_requests_total",
		"API requests observed since process start.", "counter",
		strconv.FormatInt(snapshot.APIRequestsTotal, 10))
	writeMetric("quickdesk_signaling_api_requests_today",
		"API requests observed today in UTC.", "gauge",
		strconv.FormatInt(snapshot.APIRequestsToday, 10))
	// Per-route counters are what alert rules can actually be written against:
	// a total says "traffic", a route plus status says "which endpoint is
	// failing". The families are always declared, even with no series yet, so a
	// rule file referring to them is valid from the first scrape.
	fmt.Fprintf(&b, "# HELP quickdesk_signaling_http_requests_by_route_total Completed /v1 requests, labelled by the matched route pattern, method and status code.\n")
	fmt.Fprintf(&b, "# TYPE quickdesk_signaling_http_requests_by_route_total counter\n")
	for _, row := range snapshot.HTTPRouteRequests {
		fmt.Fprintf(&b, "quickdesk_signaling_http_requests_by_route_total{route=%s,method=%s,status=%s} %d\n",
			prometheusLabelValue(row.Route),
			prometheusLabelValue(row.Method),
			prometheusLabelValue(strconv.Itoa(row.Status)),
			row.Count)
	}
	writeMetric("quickdesk_signaling_websocket_connections",
		"Current websocket connections across signaling and event streams.", "gauge",
		strconv.Itoa(snapshot.WebSocketConnections))
	writeMetric("quickdesk_signaling_signal_hosts",
		"Current connected signaling hosts.", "gauge",
		strconv.Itoa(snapshot.SignalHosts))
	writeMetric("quickdesk_signaling_signal_clients",
		"Current connected signaling clients.", "gauge",
		strconv.Itoa(snapshot.SignalClients))
	writeMetric("quickdesk_signaling_event_streams",
		"Current connected event streams.", "gauge",
		strconv.Itoa(snapshot.EventStreams))
	writeMetric("quickdesk_signaling_go_goroutines",
		"Current number of goroutines.", "gauge",
		strconv.Itoa(runtime.NumGoroutine()))
	writeMetric("quickdesk_signaling_go_mem_alloc_bytes",
		"Current Go heap allocation in bytes.", "gauge",
		strconv.FormatUint(mem.Alloc, 10))
	writeMetric("quickdesk_signaling_go_mem_sys_bytes",
		"Current Go runtime memory obtained from the OS in bytes.", "gauge",
		strconv.FormatUint(mem.Sys, 10))

	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, b.String())
}

// validMetricsToken checks a Prometheus bearer token without leaking length or
// prefix through an early-exit string comparison.
func validMetricsToken(header, want string) bool {
	const prefix = "Bearer "
	if want == "" || !strings.HasPrefix(header, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// prometheusLabelValue quotes a label value the way the text exposition format
// expects: backslash, double quote and newline are escaped and nothing else is.
// (strconv.Quote would additionally escape non-ASCII as \uXXXX, which is Go
// syntax rather than Prometheus syntax.)
func prometheusLabelValue(value string) string {
	escaped := strings.NewReplacer(
		`\`, `\\`,
		"\n", `\n`,
		`"`, `\"`,
	).Replace(value)
	return `"` + escaped + `"`
}
