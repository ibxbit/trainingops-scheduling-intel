package security

import (
	"encoding/json"
	"io"
	"net/http"
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
			entry := map[string]any{
				"ts":         time.Now().UTC().Format(time.RFC3339),
				"method":     req.Method,
				"path":       req.URL.Path,
				"status":     res.Status,
				"latency_ms": time.Since(start).Milliseconds(),
				"remote_ip":  c.RealIP(),
				"headers":    maskHeaders(req.Header),
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

func maskHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) == 0 {
			continue
		}
		lk := strings.ToLower(k)
		switch lk {
		case "authorization", "cookie", "set-cookie", "x-api-key":
			out[k] = "[REDACTED]"
		default:
			out[k] = maskSensitive(v[0])
		}
	}
	return out
}

func maskSensitive(input string) string {
	s := strings.ToLower(input)
	if strings.Contains(s, "password") || strings.Contains(s, "token") || strings.Contains(s, "secret") {
		return "[REDACTED]"
	}
	return input
}
