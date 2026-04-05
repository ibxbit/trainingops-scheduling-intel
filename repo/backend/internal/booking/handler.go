package booking

import (
	"errors"
	"net/http"

	"trainingops/backend/internal/access"
	"trainingops/backend/internal/calendar"
	"trainingops/backend/internal/rbac"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc         *Service
	calendarSvc *calendar.Service
}

func NewHandler(svc *Service, calendarSvc *calendar.Service) *Handler {
	return &Handler{svc: svc, calendarSvc: calendarSvc}
}

type holdRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}

type reasonRequest struct {
	Reason string `json:"reason"`
}

type rescheduleRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}

func (h *Handler) Hold(c echo.Context) error {
	tenantID, userID, ok := scopedIdentity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req holdRequest
	if err := c.Bind(&req); err != nil || req.SessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	b, err := h.svc.Hold(c.Request().Context(), tenantID, userID, req.SessionID, req.Reason)
	if err != nil {
		reason, alternatives := h.conflictDetails(c, tenantID, req.SessionID)
		if errors.Is(err, ErrCapacityReached) {
			return c.JSON(http.StatusConflict, map[string]any{"error": "capacity reached", "reason": reason, "alternatives": alternatives})
		}
		if errors.Is(err, ErrRoomOccupied) {
			return c.JSON(http.StatusConflict, map[string]any{"error": "room occupied", "reason": reason, "alternatives": alternatives})
		}
		if errors.Is(err, ErrInstructorBusy) {
			return c.JSON(http.StatusConflict, map[string]any{"error": "instructor unavailable", "reason": reason, "alternatives": alternatives})
		}
		if errors.Is(err, ErrOutsideTerm) {
			return c.JSON(http.StatusConflict, map[string]any{"error": "outside academic calendar", "reason": reason, "alternatives": alternatives})
		}
		if errors.Is(err, ErrBlackoutDate) {
			return c.JSON(http.StatusConflict, map[string]any{"error": "blackout date", "reason": reason, "alternatives": alternatives})
		}
		if errors.Is(err, ErrOutsideSlot) {
			return c.JSON(http.StatusConflict, map[string]any{"error": "outside allowed slot", "reason": reason, "alternatives": alternatives})
		}
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
		}
		return c.JSON(http.StatusConflict, map[string]any{"error": "unable to place hold", "reason": reason, "alternatives": alternatives})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"data": map[string]any{
			"booking_id":       b.BookingID,
			"state":            b.State,
			"hold_expires_at":  b.HoldExpiresAt,
			"reschedule_count": b.RescheduleCount,
		},
	})
}

func (h *Handler) Confirm(c echo.Context) error {
	tenantID, userID, ok := scopedIdentity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	bookingID := c.Param("booking_id")
	if bookingID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "booking_id is required"})
	}

	var req reasonRequest
	_ = c.Bind(&req)

	roles, _ := c.Get(access.ContextRoles).([]rbac.Role)
	err := h.svc.Confirm(c.Request().Context(), tenantID, userID, roles, bookingID, req.Reason)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "booking not found"})
		}
		if errors.Is(err, ErrInvalidState) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "booking not in held state"})
		}
		return c.JSON(http.StatusConflict, map[string]string{"error": "unable to confirm booking"})
	}

	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "confirmed"}})
}

func (h *Handler) Reschedule(c echo.Context) error {
	tenantID, userID, ok := scopedIdentity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	bookingID := c.Param("booking_id")
	if bookingID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "booking_id is required"})
	}

	var req rescheduleRequest
	if err := c.Bind(&req); err != nil || req.SessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_id is required"})
	}

	roles, _ := c.Get(access.ContextRoles).([]rbac.Role)
	err := h.svc.Reschedule(c.Request().Context(), tenantID, userID, roles, bookingID, req.SessionID, req.Reason)
	if err != nil {
		reason, alternatives := h.conflictDetails(c, tenantID, req.SessionID)
		switch {
		case errors.Is(err, ErrNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "booking or session not found"})
		case errors.Is(err, ErrRoomOccupied):
			return c.JSON(http.StatusConflict, map[string]any{"error": "room occupied", "reason": reason, "alternatives": alternatives})
		case errors.Is(err, ErrInstructorBusy):
			return c.JSON(http.StatusConflict, map[string]any{"error": "instructor unavailable", "reason": reason, "alternatives": alternatives})
		case errors.Is(err, ErrOutsideTerm):
			return c.JSON(http.StatusConflict, map[string]any{"error": "outside academic calendar", "reason": reason, "alternatives": alternatives})
		case errors.Is(err, ErrBlackoutDate):
			return c.JSON(http.StatusConflict, map[string]any{"error": "blackout date", "reason": reason, "alternatives": alternatives})
		case errors.Is(err, ErrOutsideSlot):
			return c.JSON(http.StatusConflict, map[string]any{"error": "outside allowed slot", "reason": reason, "alternatives": alternatives})
		case errors.Is(err, ErrRescheduleLimit):
			return c.JSON(http.StatusConflict, map[string]string{"error": "reschedule limit exceeded"})
		case errors.Is(err, ErrCapacityReached):
			return c.JSON(http.StatusConflict, map[string]any{"error": "capacity reached", "reason": reason, "alternatives": alternatives})
		case errors.Is(err, ErrInvalidState):
			return c.JSON(http.StatusConflict, map[string]string{"error": "booking is not reschedulable"})
		default:
			return c.JSON(http.StatusConflict, map[string]any{"error": "unable to reschedule booking", "reason": reason, "alternatives": alternatives})
		}
	}

	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "rescheduled"}})
}

func (h *Handler) Cancel(c echo.Context) error {
	tenantID, userID, ok := scopedIdentity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	bookingID := c.Param("booking_id")
	if bookingID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "booking_id is required"})
	}

	var req reasonRequest
	_ = c.Bind(&req)

	roles, _ := c.Get(access.ContextRoles).([]rbac.Role)
	err := h.svc.Cancel(c.Request().Context(), tenantID, userID, roles, bookingID, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "booking not found"})
		case errors.Is(err, ErrCancellationCutoff):
			return c.JSON(http.StatusConflict, map[string]string{"error": "cancellation cutoff exceeded"})
		case errors.Is(err, ErrInvalidState):
			return c.JSON(http.StatusConflict, map[string]string{"error": "booking is not cancelable"})
		default:
			return c.JSON(http.StatusConflict, map[string]string{"error": "unable to cancel booking"})
		}
	}

	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "canceled"}})
}

func (h *Handler) CheckIn(c echo.Context) error {
	tenantID, actorUserID, ok := scopedIdentity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	bookingID := c.Param("booking_id")
	if bookingID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "booking_id is required"})
	}

	var req reasonRequest
	_ = c.Bind(&req)

	err := h.svc.CheckIn(c.Request().Context(), tenantID, actorUserID, bookingID, req.Reason)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "booking not found"})
		}
		if errors.Is(err, ErrInvalidState) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "booking is not check-in eligible"})
		}
		return c.JSON(http.StatusConflict, map[string]string{"error": "unable to check in"})
	}

	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "checked_in"}})
}

func scopedIdentity(c echo.Context) (string, string, bool) {
	tenantID, okTenant := c.Get(access.ContextTenantID).(string)
	userID, okUser := c.Get(access.ContextUserID).(string)
	if !okTenant || !okUser || tenantID == "" || userID == "" {
		return "", "", false
	}
	return tenantID, userID, true
}

func (h *Handler) conflictDetails(c echo.Context, tenantID, sessionID string) (calendar.AvailabilityReason, []calendar.Alternative) {
	if h.calendarSvc == nil || sessionID == "" {
		return "", nil
	}
	reason, alternatives, err := h.calendarSvc.CheckAvailability(c.Request().Context(), tenantID, sessionID)
	if err != nil {
		return "", nil
	}
	return reason, alternatives
}
