package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSanitizePathRedactsShareToken(t *testing.T) {
	path := "/api/v1/content/share/super-secret-token/download"
	got := sanitizePath(path)
	if got != "/api/v1/content/share/[REDACTED]/download" {
		t.Fatalf("sanitizePath() = %q", got)
	}
}

func TestAnonymizeIPStableHash(t *testing.T) {
	a := anonymizeIP("192.168.1.24")
	b := anonymizeIP("192.168.1.24")
	if a == "" || !strings.HasPrefix(a, "iphash:") {
		t.Fatalf("unexpected hash output: %q", a)
	}
	if a != b {
		t.Fatalf("expected stable hash, got %q and %q", a, b)
	}
}

func TestRequestLogMiddlewareRedactsPathAndIP(t *testing.T) {
	e := echo.New()
	body := &bytes.Buffer{}
	logger := NewSecureLogger(body)
	e.Use(RequestLogMiddleware(logger))
	e.GET("/api/v1/content/share/:token/download", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/content/share/abc123/download", nil)
	req.Header.Set(echo.HeaderXRequestID, "req-1")
	req.RemoteAddr = "10.0.0.42:9090"
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	line := strings.TrimSpace(body.String())
	if line == "" {
		t.Fatal("expected log output")
	}

	entry := map[string]any{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if entry["path"] != "/api/v1/content/share/[REDACTED]/download" {
		t.Fatalf("unexpected path: %#v", entry["path"])
	}
	if entry["route"] != "/api/v1/content/share/:token/download" {
		t.Fatalf("unexpected route: %#v", entry["route"])
	}
	if ip, _ := entry["client_ip"].(string); strings.Contains(ip, "10.0.0.42") {
		t.Fatalf("ip should be masked, got %q", ip)
	}
}
