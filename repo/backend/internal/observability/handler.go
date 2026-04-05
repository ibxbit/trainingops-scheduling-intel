package observability

import (
	"net/http"
	"strconv"
	"time"

	"trainingops/backend/internal/access"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) WorkflowLogs(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, err := h.svc.WorkflowLogs(c.Request().Context(), tenantID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

type scrapeErrRequest struct {
	SourceName   string         `json:"source_name"`
	ErrorCode    string         `json:"error_code"`
	ErrorMessage string         `json:"error_message"`
	Metadata     map[string]any `json:"metadata"`
}

func (h *Handler) RecordScrapingError(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req scrapeErrRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if err := h.svc.RecordScrapingError(c.Request().Context(), tenantID, req.SourceName, req.ErrorCode, req.ErrorMessage, req.Metadata); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]string{"status": "recorded"}})
}

type detectAnomalyRequest struct {
	Date string `json:"date"`
}

func (h *Handler) DetectAnomalies(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req detectAnomalyRequest
	_ = c.Bind(&req)
	n, err := h.svc.DetectAnomalies(c.Request().Context(), tenantID, req.Date)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "detect failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]int{"detected": n}})
}

func (h *Handler) ListAnomalies(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	date := c.QueryParam("date")
	items, err := h.svc.ListAnomalies(c.Request().Context(), tenantID, date, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

type createScheduleRequest struct {
	Name         string  `json:"name"`
	Format       string  `json:"format"`
	Frequency    string  `json:"frequency"`
	OutputFolder string  `json:"output_folder"`
	NextRunAt    *string `json:"next_run_at"`
}

func (h *Handler) CreateSchedule(c echo.Context) error {
	tenantID, okTenant := c.Get(access.ContextTenantID).(string)
	userID, okUser := c.Get(access.ContextUserID).(string)
	if !okTenant || !okUser || tenantID == "" || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req createScheduleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	var next *time.Time
	if req.NextRunAt != nil && *req.NextRunAt != "" {
		v, err := time.Parse(time.RFC3339, *req.NextRunAt)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid next_run_at"})
		}
		u := v.UTC()
		next = &u
	}
	s, err := h.svc.CreateSchedule(c.Request().Context(), tenantID, userID, req.Name, req.Format, req.Frequency, req.OutputFolder, next)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": s})
}

func (h *Handler) RunDueSchedules(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	items, err := h.svc.RunDueSchedules(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "run failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

type runNowRequest struct {
	ReportDate string `json:"report_date"`
}

func (h *Handler) RunScheduleNow(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	scheduleID := c.Param("schedule_id")
	var req runNowRequest
	_ = c.Bind(&req)
	exp, err := h.svc.RunScheduleNow(c.Request().Context(), tenantID, scheduleID, req.ReportDate)
	if err != nil {
		if err == ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "schedule not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "run failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": exp})
}

func (h *Handler) ListExports(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, err := h.svc.ListExports(c.Request().Context(), tenantID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}
