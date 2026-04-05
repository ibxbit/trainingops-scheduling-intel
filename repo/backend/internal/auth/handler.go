package auth

import (
	"net/http"
	"time"

	"trainingops/backend/internal/config"
	"trainingops/backend/internal/rbac"
	"trainingops/backend/internal/security"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type loginRequest struct {
	TenantSlug string `json:"tenant_slug"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

func (h *Handler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.TenantSlug == "" || req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_slug, username, and password are required"})
	}

	cookieValue, expiresAt, err := h.svc.Login(c.Request().Context(), req.TenantSlug, req.Username, req.Password, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		case ErrUserLocked:
			return c.JSON(http.StatusLocked, map[string]string{"error": "account locked for 15 minutes after 5 failed attempts"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "login failed"})
		}
	}

	setSessionCookie(c, h.svc.cfg, cookieValue, expiresAt)
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "authenticated"}})
}

func (h *Handler) Logout(c echo.Context) error {
	tenantID, raw := readSessionToken(c, h.svc.cfg.SessionCookieName)
	_ = h.svc.Logout(c.Request().Context(), tenantID, raw)
	clearSessionCookie(c, h.svc.cfg)
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "logged_out"}})
}

func (h *Handler) Me(c echo.Context) error {
	tenantID, okTenant := c.Get("tenant_id").(string)
	userID, okUser := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]rbac.Role)
	if !okTenant || !okUser || tenantID == "" || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}
	roleNames := make([]string, 0, len(roles))
	for _, role := range roles {
		roleNames = append(roleNames, string(role))
	}
	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
			"roles":     roleNames,
		},
	})
}

func (h *Handler) ValidateUpload(c echo.Context) error {
	checksum := c.FormValue("checksum_sha256")
	f, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}

	actual, err := security.ValidateUpload(f, h.svc.cfg.AllowedUploadFormats, checksum)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]string{
			"checksum_sha256": actual,
		},
	})
}

func setSessionCookie(c echo.Context, cfg *config.Config, value string, expiresAt time.Time) {
	cookie := new(http.Cookie)
	cookie.Name = cfg.SessionCookieName
	cookie.Value = value
	cookie.Expires = expiresAt
	cookie.HttpOnly = true
	cookie.Path = "/"
	cookie.SameSite = http.SameSiteStrictMode
	cookie.Secure = cfg.SessionSecureCookie
	cookie.MaxAge = int(time.Until(expiresAt).Seconds())
	c.SetCookie(cookie)
}

func clearSessionCookie(c echo.Context, cfg *config.Config) {
	cookie := new(http.Cookie)
	cookie.Name = cfg.SessionCookieName
	cookie.Value = ""
	cookie.Path = "/"
	cookie.Expires = time.Unix(0, 0)
	cookie.MaxAge = -1
	cookie.HttpOnly = true
	cookie.Secure = cfg.SessionSecureCookie
	cookie.SameSite = http.SameSiteStrictMode
	c.SetCookie(cookie)
}

func readSessionToken(c echo.Context, cookieName string) (string, string) {
	cookie, err := c.Cookie(cookieName)
	if err != nil {
		return "", ""
	}
	return ParseCookieValue(cookie.Value)
}
