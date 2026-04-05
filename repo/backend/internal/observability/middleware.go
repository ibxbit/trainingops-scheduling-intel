package observability

import (
	"strings"
	"time"

	"trainingops/backend/internal/access"

	"github.com/labstack/echo/v4"
)

func WorkflowLogMiddleware(svc *Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			method := c.Request().Method
			if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
				return err
			}

			tenantID, _ := c.Get(access.ContextTenantID).(string)
			if tenantID == "" {
				return err
			}
			userID, _ := c.Get(access.ContextUserID).(string)
			var userPtr *string
			if userID != "" {
				userPtr = &userID
			}

			status := c.Response().Status
			outcome := "success"
			if status >= 400 || err != nil {
				outcome = "failed"
			}

			workflowName := inferWorkflowName(c.Path())
			resourceID := inferResourceID(c)
			latency := int(time.Since(start).Milliseconds())

			svc.LogWorkflow(c.Request().Context(), tenantID, userPtr, workflowName, resourceID, outcome, status, latency, map[string]any{
				"method": method,
				"path":   c.Path(),
			})
			return err
		}
	}
}

func inferWorkflowName(path string) string {
	p := strings.Trim(path, "/")
	if p == "" {
		return "system.unknown"
	}
	parts := strings.Split(p, "/")
	if len(parts) >= 3 && parts[0] == "api" {
		return parts[2] + ".request"
	}
	if len(parts) > 0 {
		return parts[0] + ".request"
	}
	return "system.unknown"
}

func inferResourceID(c echo.Context) string {
	keys := []string{"booking_id", "session_id", "document_id", "task_id", "plan_id", "milestone_id", "schedule_id", "upload_id"}
	for _, k := range keys {
		if v := c.Param(k); v != "" {
			return v
		}
	}
	return ""
}
