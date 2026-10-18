package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// alertRules mirrors the parts of the Prometheus rule file this repository
// depends on. Unknown fields are ignored, so annotations can grow freely.
type alertRules struct {
	Groups []struct {
		Name  string `yaml:"name"`
		Rules []struct {
			Alert       string            `yaml:"alert"`
			Expr        string            `yaml:"expr"`
			For         string            `yaml:"for"`
			Labels      map[string]string `yaml:"labels"`
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"rules"`
	} `yaml:"groups"`
}

func loadAlertRules(t *testing.T) alertRules {
	t.Helper()
	path := filepath.Join("..", "..", "deploy", "prometheus", "quickdesk-signaling.rules.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rules: %v", err)
	}
	var rules alertRules
	if err := yaml.Unmarshal(raw, &rules); err != nil {
		t.Fatalf("parse rules: %v", err)
	}
	if len(rules.Groups) == 0 {
		t.Fatalf("no rule groups in %s", path)
	}
	return rules
}

// renderMetricsTestExposition returns the text /metrics would serve, so a test
// can compare it against the rule file instead of against a hand-kept list.
func renderMetricsTestExposition(t *testing.T) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	metrics := service.NewMetricsService()
	metrics.RecordHTTPRequest("GET", "/v1/me/devices", "/v1/me/devices", 200)
	h := NewMetricsHandler(metrics, "test-version", "metrics-secret")
	router := gin.New()
	router.GET("/metrics", h.Prometheus)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer metrics-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", rec.Code)
	}
	return rec.Body.String()
}

// Every metric an alert rule refers to has to exist in the exposition, and it
// has to be declared there even before traffic arrives. A rule pointing at a
// metric nobody emits is silent, which is the worst possible failure mode for an
// alert.
func TestAlertRulesReferenceExportedMetrics(t *testing.T) {
	rules := loadAlertRules(t)
	exposition := renderMetricsTestExposition(t)

	namePattern := regexp.MustCompile(`quickdesk_signaling_[a-z0-9_]+`)
	seen := 0
	for _, group := range rules.Groups {
		for _, rule := range group.Rules {
			for _, name := range namePattern.FindAllString(rule.Expr, -1) {
				seen++
				if !strings.Contains(exposition, "# TYPE "+name+" ") {
					t.Errorf("rule %q refers to %s, which /metrics does not declare", rule.Alert, name)
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("no metric names found in the rule file; the regexp or the file shape changed")
	}
}

// The rules are configuration an operator copies into Prometheus, so the fields
// Prometheus requires have to be present, and every rule needs a severity for
// routing plus a summary for the notification.
func TestAlertRulesAreWellFormed(t *testing.T) {
	rules := loadAlertRules(t)
	count := 0
	for _, group := range rules.Groups {
		if strings.TrimSpace(group.Name) == "" {
			t.Fatal("rule group without a name")
		}
		for _, rule := range group.Rules {
			count++
			if rule.Alert == "" {
				t.Error("rule without an alert name")
			}
			if strings.TrimSpace(rule.Expr) == "" {
				t.Errorf("rule %s has no expression", rule.Alert)
			}
			if strings.TrimSpace(rule.For) == "" {
				t.Errorf("rule %s has no `for`, so it would fire on a single sample", rule.Alert)
			}
			if rule.Labels["severity"] == "" {
				t.Errorf("rule %s has no severity label", rule.Alert)
			}
			if strings.TrimSpace(rule.Annotations["summary"]) == "" {
				t.Errorf("rule %s has no summary annotation", rule.Alert)
			}
		}
	}
	if count == 0 {
		t.Fatal("no rules in the rule file")
	}
}
