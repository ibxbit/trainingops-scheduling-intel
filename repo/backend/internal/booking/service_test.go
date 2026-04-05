package booking

import (
	"testing"

	"trainingops/backend/internal/rbac"
)

func TestCanManageTenantBookings(t *testing.T) {
	tests := []struct {
		name  string
		roles []rbac.Role
		want  bool
	}{
		{name: "administrator can manage", roles: []rbac.Role{rbac.RoleAdministrator}, want: true},
		{name: "program coordinator can manage", roles: []rbac.Role{rbac.RoleProgramCoordinator}, want: true},
		{name: "learner cannot manage", roles: []rbac.Role{rbac.RoleLearner}, want: false},
		{name: "instructor cannot manage", roles: []rbac.Role{rbac.RoleInstructor}, want: false},
		{name: "mixed with manager can manage", roles: []rbac.Role{rbac.RoleLearner, rbac.RoleProgramCoordinator}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := canManageTenantBookings(tc.roles)
			if got != tc.want {
				t.Fatalf("canManageTenantBookings() = %v, want %v", got, tc.want)
			}
		})
	}
}
