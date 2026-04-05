package access

import (
	"database/sql"
	"net/http"
	"time"

	"trainingops/backend/internal/auth"
	"trainingops/backend/internal/config"
	"trainingops/backend/internal/dbctx"
	"trainingops/backend/internal/rbac"

	"github.com/labstack/echo/v4"
)

const (
	ContextTenantID = "tenant_id"
	ContextUserID   = "user_id"
	ContextRoles    = "roles"
)

type Middleware struct {
	authService *auth.Service
	cfg         *config.Config
	db          *sql.DB
}

func NewMiddleware(authService *auth.Service, cfg *config.Config, db *sql.DB) *Middleware {
	return &Middleware{authService: authService, cfg: cfg, db: db}
}

func (m *Middleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(m.cfg.SessionCookieName)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing session"})
		}

		tenantID, rawToken := auth.ParseCookieValue(cookie.Value)
		if rawToken == "" || tenantID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid session"})
		}

		newToken, session, rotated, err := m.authService.ValidateAndRotate(c.Request().Context(), tenantID, rawToken)
		if err != nil {
			clearSessionCookie(c, m.cfg)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired session"})
		}

		if rotated {
			setSessionCookie(c, m.cfg, newToken, session.ExpiresAt)
		}

		c.Set(ContextTenantID, session.TenantID)
		c.Set(ContextUserID, session.UserID)
		c.Set(ContextRoles, session.Roles)
		return next(c)
	}
}

func (m *Middleware) TenantScope(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tenantID, ok := c.Get(ContextTenantID).(string)
		if !ok || tenantID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant scope unavailable"})
		}

		if routeTenant := c.Param("tenant_id"); routeTenant != "" && routeTenant != tenantID {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "cross-tenant access denied"})
		}

		return next(c)
	}
}

func (m *Middleware) BindTenantDB(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tenantID, ok := c.Get(ContextTenantID).(string)
		if !ok || tenantID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant scope unavailable"})
		}

		conn, err := m.db.Conn(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database connection unavailable"})
		}
		defer conn.Close()

		if _, err := conn.ExecContext(c.Request().Context(), `SELECT set_config('app.tenant_id', $1, false)`, tenantID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant db scope setup failed"})
		}
		defer conn.ExecContext(c.Request().Context(), `RESET app.tenant_id`)

		ctx := dbctx.WithConn(c.Request().Context(), conn)
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}

func (m *Middleware) RequireRoles(allowed ...rbac.Role) echo.MiddlewareFunc {
	allowedSet := make(map[rbac.Role]struct{}, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			hasRoles, ok := c.Get(ContextRoles).([]rbac.Role)
			if !ok || len(hasRoles) == 0 {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "role required"})
			}

			for _, role := range hasRoles {
				if _, ok := allowedSet[role]; ok {
					return next(c)
				}
			}
			return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient role"})
		}
	}
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
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)
	cookie.HttpOnly = true
	cookie.Secure = cfg.SessionSecureCookie
	cookie.SameSite = http.SameSiteStrictMode
	c.SetCookie(cookie)
}
