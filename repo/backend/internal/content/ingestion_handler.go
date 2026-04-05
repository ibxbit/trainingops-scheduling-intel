package content

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type createIngestionSourceRequest struct {
	Name                    string `json:"name"`
	BaseURL                 string `json:"base_url"`
	ScheduleIntervalMinutes int    `json:"schedule_interval_minutes"`
	ScheduleJitterSeconds   int    `json:"schedule_jitter_seconds"`
	RateLimitPerMinute      int    `json:"rate_limit_per_minute"`
	RequestTimeoutSeconds   int    `json:"request_timeout_seconds"`
}

func (h *Handler) CreateIngestionSource(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req createIngestionSourceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	source, err := h.svc.CreateIngestionSource(
		c.Request().Context(),
		tenantID,
		userID,
		req.Name,
		req.BaseURL,
		req.ScheduleIntervalMinutes,
		req.ScheduleJitterSeconds,
		req.RateLimitPerMinute,
		req.RequestTimeoutSeconds,
	)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": source})
}

func (h *Handler) ListIngestionSources(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, err := h.svc.ListIngestionSources(c.Request().Context(), tenantID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

type ingestionProxyRequest struct {
	ProxyURL string `json:"proxy_url"`
}

func (h *Handler) AddIngestionProxy(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req ingestionProxyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := h.svc.AddIngestionProxy(c.Request().Context(), tenantID, req.ProxyURL); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]string{"status": "proxy_added"}})
}

type ingestionUserAgentRequest struct {
	UserAgent string `json:"user_agent"`
}

func (h *Handler) AddIngestionUserAgent(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req ingestionUserAgentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := h.svc.AddIngestionUserAgent(c.Request().Context(), tenantID, req.UserAgent); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]string{"status": "user_agent_added"}})
}

func (h *Handler) RunDueIngestion(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	maxSources, _ := strconv.Atoi(c.QueryParam("max_sources"))
	runs, err := h.svc.RunDueIngestion(c.Request().Context(), tenantID, maxSources)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "run failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": runs})
}

func (h *Handler) RunIngestionNow(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	sourceID := c.Param("source_id")
	run, err := h.svc.RunIngestionNow(c.Request().Context(), tenantID, sourceID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "source not found"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": run})
}

func (h *Handler) ListIngestionRuns(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	items, err := h.svc.ListIngestionRuns(c.Request().Context(), tenantID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": items})
}

type manualReviewRequest struct {
	Approve bool   `json:"approve"`
	Reason  string `json:"reason"`
}

func (h *Handler) SetIngestionManualReview(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	sourceID := c.Param("source_id")
	var req manualReviewRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := h.svc.SetIngestionManualReview(c.Request().Context(), tenantID, sourceID, userID, req.Approve, req.Reason); err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "source not found"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	status := "paused"
	if req.Approve {
		status = "approved"
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": status}})
}
