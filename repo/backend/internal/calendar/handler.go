package calendar

import (
	"errors"
	"net/http"

	"trainingops/backend/internal/access"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Availability(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	sessionID := c.Param("session_id")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	reason, alts, err := h.svc.CheckAvailability(c.Request().Context(), tenantID, sessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "availability check failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]any{"reason": reason, "alternatives": alts}})
}

type timeSlotRuleRequest struct {
	RoomID      *string `json:"room_id"`
	Weekday     int     `json:"weekday"`
	SlotStart   string  `json:"slot_start"`
	SlotEnd     string  `json:"slot_end"`
	IsActive    bool    `json:"is_active"`
	LockVersion int     `json:"lock_version"`
}

func (h *Handler) CreateTimeSlotRule(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req timeSlotRuleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	id, err := h.svc.CreateTimeSlotRule(c.Request().Context(), tenantID, TimeSlotRule{
		RoomID:    req.RoomID,
		Weekday:   req.Weekday,
		SlotStart: req.SlotStart,
		SlotEnd:   req.SlotEnd,
		IsActive:  req.IsActive,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "create rule failed"})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]string{"rule_id": id}})
}

func (h *Handler) UpdateTimeSlotRule(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	ruleID := c.Param("rule_id")
	var req timeSlotRuleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	err := h.svc.UpdateTimeSlotRule(c.Request().Context(), tenantID, ruleID, TimeSlotRule{
		RoomID:      req.RoomID,
		Weekday:     req.Weekday,
		SlotStart:   req.SlotStart,
		SlotEnd:     req.SlotEnd,
		IsActive:    req.IsActive,
		LockVersion: req.LockVersion,
	})
	if err != nil {
		if errors.Is(err, ErrVersionConflict) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "version conflict"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "update rule failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "updated"}})
}

type blackoutRequest struct {
	RoomID       *string `json:"room_id"`
	BlackoutDate string  `json:"blackout_date"`
	Reason       string  `json:"reason"`
	IsActive     bool    `json:"is_active"`
	LockVersion  int     `json:"lock_version"`
}

func (h *Handler) CreateBlackoutDate(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req blackoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	id, err := h.svc.CreateBlackoutDate(c.Request().Context(), tenantID, BlackoutDate{
		RoomID:       req.RoomID,
		BlackoutDate: req.BlackoutDate,
		Reason:       req.Reason,
		IsActive:     req.IsActive,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "create blackout failed"})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]string{"blackout_id": id}})
}

func (h *Handler) UpdateBlackoutDate(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	blackoutID := c.Param("blackout_id")
	var req blackoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	err := h.svc.UpdateBlackoutDate(c.Request().Context(), tenantID, blackoutID, BlackoutDate{
		RoomID:       req.RoomID,
		BlackoutDate: req.BlackoutDate,
		Reason:       req.Reason,
		IsActive:     req.IsActive,
		LockVersion:  req.LockVersion,
	})
	if err != nil {
		if errors.Is(err, ErrVersionConflict) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "version conflict"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "update blackout failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "updated"}})
}

type termRequest struct {
	Name        string `json:"name"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	IsActive    bool   `json:"is_active"`
	LockVersion int    `json:"lock_version"`
}

func (h *Handler) CreateAcademicTerm(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req termRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	id, err := h.svc.CreateAcademicTerm(c.Request().Context(), tenantID, AcademicTerm{
		Name:      req.Name,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		IsActive:  req.IsActive,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "create term failed"})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": map[string]string{"term_id": id}})
}

func (h *Handler) UpdateAcademicTerm(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	termID := c.Param("term_id")
	var req termRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	err := h.svc.UpdateAcademicTerm(c.Request().Context(), tenantID, termID, AcademicTerm{
		Name:        req.Name,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		IsActive:    req.IsActive,
		LockVersion: req.LockVersion,
	})
	if err != nil {
		if errors.Is(err, ErrVersionConflict) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "version conflict"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "update term failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "updated"}})
}
