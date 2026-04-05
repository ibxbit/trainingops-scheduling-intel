package dashboard

import (
	"errors"
	"net/http"
	"strconv"

	"trainingops/backend/internal/access"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Overview(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	metricDate := c.QueryParam("date")
	out, err := h.svc.Overview(c.Request().Context(), tenantID, metricDate)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "dashboard not precomputed for date"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "dashboard load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

type refreshRequest struct {
	Date string `json:"date"`
}

func (h *Handler) Refresh(c echo.Context) error {
	tenantID, okTenant := c.Get(access.ContextTenantID).(string)
	userID, okUser := c.Get(access.ContextUserID).(string)
	if !okTenant || !okUser || tenantID == "" || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req refreshRequest
	_ = c.Bind(&req)
	refreshID, err := h.svc.Refresh(c.Request().Context(), tenantID, userID, req.Date)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "refresh failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"refresh_id": refreshID}})
}

type featureBatchRequest struct {
	Date string `json:"date"`
}

func (h *Handler) RunNightlyFeatureBatch(c echo.Context) error {
	tenantID, okTenant := c.Get(access.ContextTenantID).(string)
	userID, okUser := c.Get(access.ContextUserID).(string)
	if !okTenant || !okUser || tenantID == "" || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req featureBatchRequest
	_ = c.Bind(&req)
	ids, err := h.svc.RunNightlyFeatureBatch(c.Request().Context(), tenantID, userID, req.Date)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "feature batch failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]any{"batch_ids": ids}})
}

func (h *Handler) LearnerFeatures(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	windowDays, _ := strconv.Atoi(c.QueryParam("window_days"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	date := c.QueryParam("date")
	segment := c.QueryParam("segment")
	items, err := h.svc.LearnerFeatures(c.Request().Context(), tenantID, date, windowDays, limit, segment)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) CohortFeatures(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	windowDays, _ := strconv.Atoi(c.QueryParam("window_days"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	date := c.QueryParam("date")
	items, err := h.svc.CohortFeatures(c.Request().Context(), tenantID, date, windowDays, limit)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) ReportingMetrics(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	windowDays, _ := strconv.Atoi(c.QueryParam("window_days"))
	date := c.QueryParam("date")
	items, err := h.svc.ReportingMetrics(c.Request().Context(), tenantID, date, windowDays)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) TodaySessions(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	metricDate := c.QueryParam("date")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, err := h.svc.TodaySessions(c.Request().Context(), tenantID, metricDate, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "today sessions load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}
