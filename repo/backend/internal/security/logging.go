package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type SecureLogger struct {
	out io.Writer
}

func NewSecureLogger(out io.Writer) *SecureLogger {
	return &SecureLogger{out: out}
}

func RequestLogMiddleware(logger *SecureLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			req := c.Request()
			res := c.Response()
			route := c.Path()
			if route == "" {
				route = sanitizePath(req.URL.Path)
			}
			entry := map[string]any{
				"ts":         time.Now().UTC().Format(time.RFC3339),
				"method":     req.Method,
				"path":       sanitizePath(req.URL.Path),
				"route":      sanitizePath(route),
				"status":     res.Status,
				"latency_ms": time.Since(start).Milliseconds(),
				"request_id": res.Header().Get(echo.HeaderXRequestID),
				"client_ip":  anonymizeIP(c.RealIP()),
			}
			if tenantID, ok := c.Get("tenant_id").(string); ok && tenantID != "" {
				entry["tenant_id"] = tenantID
			}
			if userID, ok := c.Get("user_id").(string); ok && userID != "" {
				entry["user_id"] = userID
			}
			if err != nil {
				entry["error"] = maskSensitive(err.Error())
			}
			logger.log(entry)
			return err
		}
	}
}

func (l *SecureLogger) log(entry map[string]any) {
	b, _ := json.Marshal(entry)
	_, _ = l.out.Write(append(b, '\n'))
}

func maskSensitive(input string) string {
	s := strings.ToLower(input)
	if strings.Contains(s, "password") || strings.Contains(s, "token") || strings.Contains(s, "secret") {
		return "[REDACTED]"
	}
	return input
}

func sanitizePath(path string) string {
	if path == "" {
		return path
	}
	parts := strings.Split(path, "/")
	for i := range parts {
		segment := parts[i]
		if segment == "" {
			continue
		}
		if i > 0 && parts[i-1] == "share" && !strings.HasPrefix(segment, ":") {
			parts[i] = "[REDACTED]"
			continue
		}
		if strings.HasPrefix(segment, ":") {
			continue
		}
		if strings.Contains(strings.ToLower(segment), "token") {
			parts[i] = "[REDACTED]"
		}
	}
	return strings.Join(parts, "/")
}

func anonymizeIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	host := raw
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]") {
		host = strings.TrimPrefix(strings.Split(host, "]")[0], "[")
	}
	if h, _, err := net.SplitHostPort(raw); err == nil {
		host = h
	}
	sum := sha256.Sum256([]byte(host))
	return "iphash:" + hex.EncodeToString(sum[:])[:12]
}
