package admin

import (
	"errors"
	"net/http"

	"trainingops/backend/internal/access"
	"trainingops/backend/internal/rbac"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type tenantSettingsRequest struct {
	TenantSlug                  string `json:"tenant_slug"`
	TenantName                  string `json:"tenant_name"`
	AllowSelfRegistration       bool   `json:"allow_self_registration"`
	RequireMFA                  bool   `json:"require_mfa"`
	MaxActiveBookingsPerLearner int    `json:"max_active_bookings_per_learner"`
}

func (h *Handler) ListTenantSettings(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	out, err := h.svc.ListTenantSettings(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant settings load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) CreateTenantSettings(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req tenantSettingsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	out, err := h.svc.CreateTenantSettings(c.Request().Context(), TenantSettings{
		TenantID:                    tenantID,
		TenantSlug:                  req.TenantSlug,
		TenantName:                  req.TenantName,
		AllowSelfRegistration:       req.AllowSelfRegistration,
		RequireMFA:                  req.RequireMFA,
		MaxActiveBookingsPerLearner: req.MaxActiveBookingsPerLearner,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant not found"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": out})
}

func (h *Handler) UpdateTenantSettings(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	pathTenantID := c.Param("tenant_id")
	if pathTenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}
	var req tenantSettingsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	out, err := h.svc.UpdateTenantSettings(c.Request().Context(), TenantSettings{
		TenantID:                    tenantID,
		TenantSlug:                  req.TenantSlug,
		TenantName:                  req.TenantName,
		AllowSelfRegistration:       req.AllowSelfRegistration,
		RequireMFA:                  req.RequireMFA,
		MaxActiveBookingsPerLearner: req.MaxActiveBookingsPerLearner,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant not found"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

type rolePermissionUpdateRequest struct {
	Assignments []RolePermission `json:"assignments"`
}

func (h *Handler) RolePermissionMatrix(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	out, err := h.svc.RolePermissionMatrix(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "permission matrix load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) UpdateRolePermissionMatrix(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req rolePermissionUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := h.svc.UpdateRolePermissionMatrix(c.Request().Context(), tenantID, req.Assignments); err != nil {
		if errors.Is(err, ErrInvalidRole) || errors.Is(err, ErrInvalidPolicy) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid assignment payload"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "updated"}})
}

type userRoleUpdateRequest struct {
	Role rbac.Role `json:"role"`
}

func (h *Handler) ListUserRoles(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	out, err := h.svc.ListUserRoleAssignments(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user roles load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) AssignUserRole(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}
	var req userRoleUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := h.svc.AssignUserRole(c.Request().Context(), tenantID, userID, req.Role); err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		if errors.Is(err, ErrInvalidRole) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "role assignment failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "assigned"}})
}

func (h *Handler) RevokeUserRole(c echo.Context) error {
	tenantID, ok := c.Get(access.ContextTenantID).(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	userID := c.Param("user_id")
	role := rbac.Role(c.Param("role"))
	if userID == "" || role == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id and role are required"})
	}
	if err := h.svc.RevokeUserRole(c.Request().Context(), tenantID, userID, role); err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		if errors.Is(err, ErrInvalidRole) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "role revoke failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "revoked"}})
}
