package admin

import "trainingops/backend/internal/rbac"

type TenantSettings struct {
	TenantID                    string `json:"tenant_id"`
	TenantSlug                  string `json:"tenant_slug"`
	TenantName                  string `json:"tenant_name"`
	AllowSelfRegistration       bool   `json:"allow_self_registration"`
	RequireMFA                  bool   `json:"require_mfa"`
	MaxActiveBookingsPerLearner int    `json:"max_active_bookings_per_learner"`
}

type RolePermission struct {
	Role       rbac.Role `json:"role"`
	Permission string    `json:"permission"`
	Allowed    bool      `json:"allowed"`
}

type UserRoleAssignment struct {
	UserID   string      `json:"user_id"`
	Username string      `json:"username"`
	Roles    []rbac.Role `json:"roles"`
}
