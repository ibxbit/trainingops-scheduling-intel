package admin

import (
	"context"
	"errors"
	"strings"

	"trainingops/backend/internal/rbac"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListTenantSettings(ctx context.Context, tenantID string) ([]TenantSettings, error) {
	return s.repo.ListTenantSettings(ctx, tenantID)
}

func (s *Service) CreateTenantSettings(ctx context.Context, settings TenantSettings) (*TenantSettings, error) {
	if err := validateSettingsInput(settings); err != nil {
		return nil, err
	}
	return s.repo.CreateTenantSettings(ctx, settings)
}

func (s *Service) UpdateTenantSettings(ctx context.Context, settings TenantSettings) (*TenantSettings, error) {
	if err := validateSettingsInput(settings); err != nil {
		return nil, err
	}
	return s.repo.UpdateTenantSettings(ctx, settings)
}

func (s *Service) RolePermissionMatrix(ctx context.Context, tenantID string) ([]RolePermission, error) {
	return s.repo.RolePermissionMatrix(ctx, tenantID)
}

func (s *Service) UpdateRolePermissionMatrix(ctx context.Context, tenantID string, assignments []RolePermission) error {
	if len(assignments) == 0 {
		return errors.New("at least one role-permission assignment is required")
	}
	for _, assignment := range assignments {
		if !rbac.IsKnownRole(assignment.Role) {
			return ErrInvalidRole
		}
		if strings.TrimSpace(assignment.Permission) == "" {
			return ErrInvalidPolicy
		}
	}
	return s.repo.UpdateRolePermissionMatrix(ctx, tenantID, assignments)
}

func (s *Service) ListUserRoleAssignments(ctx context.Context, tenantID string) ([]UserRoleAssignment, error) {
	return s.repo.ListUserRoleAssignments(ctx, tenantID)
}

func (s *Service) AssignUserRole(ctx context.Context, tenantID, userID string, role rbac.Role) error {
	if userID == "" {
		return ErrNotFound
	}
	if !rbac.IsKnownRole(role) {
		return ErrInvalidRole
	}
	return s.repo.AssignUserRole(ctx, tenantID, userID, role)
}

func (s *Service) RevokeUserRole(ctx context.Context, tenantID, userID string, role rbac.Role) error {
	if userID == "" {
		return ErrNotFound
	}
	if !rbac.IsKnownRole(role) {
		return ErrInvalidRole
	}
	return s.repo.RevokeUserRole(ctx, tenantID, userID, role)
}

func validateSettingsInput(settings TenantSettings) error {
	if strings.TrimSpace(settings.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(settings.TenantSlug) == "" {
		return errors.New("tenant_slug is required")
	}
	if strings.TrimSpace(settings.TenantName) == "" {
		return errors.New("tenant_name is required")
	}
	if settings.MaxActiveBookingsPerLearner < 1 || settings.MaxActiveBookingsPerLearner > 20 {
		return errors.New("max_active_bookings_per_learner must be between 1 and 20")
	}
	return nil
}
