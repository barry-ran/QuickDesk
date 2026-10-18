package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

func metricsTestRouter(token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	metrics := service.NewMetricsService()
	metrics.RecordHTTPRequest("GET", "/v1/me/devices", "/v1/me/devices", 200)
	metrics.RecordHTTPRequest("POST", "/v1/devices/123456789/heartbeat", "/v1/devices/:device_id/heartbeat", 500)
	h := NewMetricsHandler(metrics, "test-version", token)
	router := gin.New()
	router.GET("/metrics", h.Prometheus)
	return router
}

func TestPrometheusMetricsRequiresBearerToken(t *testing.T) {
	router := metricsTestRouter("metrics-secret")

	for _, tc := range []struct {
		name   string
		header string
	}{
		{name: "missing", header: ""},
		{name: "wrong", header: "Bearer wrong-secret"},
		{name: "naked", header: "metrics-secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
		})
	}
}

func TestPrometheusMetricsTextExposition(t *testing.T) {
	router := metricsTestRouter("metrics-secret")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer metrics-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain", got)
	}
	for _, want := range []string{
		"quickdesk_signaling_build_info{",
		`version="test-version"`,
		"quickdesk_signaling_api_requests_total 2",
		"quickdesk_signaling_api_requests_today 2",
		"quickdesk_signaling_websocket_connections 0",
		"quickdesk_signaling_go_goroutines ",
		"# TYPE quickdesk_signaling_http_requests_by_route_total counter",
		`quickdesk_signaling_http_requests_by_route_total{route="/v1/me/devices",method="GET",status="200"} 1`,
		`quickdesk_signaling_http_requests_by_route_total{route="/v1/devices/:device_id/heartbeat",method="POST",status="500"} 1`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("metrics body missing %q:\n%s", want, rec.Body.String())
		}
	}
}

// Label values are quoted the way the exposition format expects, and only the
// three characters that format defines as escapable are escaped.
func TestPrometheusMetricsEscapesLabelValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := service.NewMetricsService()
	metrics.RecordHTTPRequest("GET", "/v1/odd", "/v1/odd\"path\\with\nnewline", 418)

	h := NewMetricsHandler(metrics, "test-version", "metrics-secret")
	router := gin.New()
	router.GET("/metrics", h.Prometheus)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer metrics-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	want := `route="/v1/odd\"path\\with\nnewline"`
	if !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("metrics body missing %s:\n%s", want, rec.Body.String())
	}
}

func TestPrometheusMetricsDisabledWithoutToken(t *testing.T) {
	router := metricsTestRouter("")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 when METRICS_TOKEN is unset", rec.Code)
	}
}
