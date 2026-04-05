package booking

import (
	"context"
	"time"

	"trainingops/backend/internal/rbac"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Hold(ctx context.Context, tenantID, learnerUserID, sessionID, reason string) (*Booking, error) {
	return s.repo.Hold(ctx, tenantID, learnerUserID, sessionID, reason, 5*time.Minute)
}

func (s *Service) Confirm(ctx context.Context, tenantID, actorUserID string, actorRoles []rbac.Role, bookingID, reason string) error {
	return s.repo.Confirm(ctx, tenantID, actorUserID, bookingID, reason, canManageTenantBookings(actorRoles))
}

func (s *Service) Reschedule(ctx context.Context, tenantID, actorUserID string, actorRoles []rbac.Role, bookingID, newSessionID, reason string) error {
	return s.repo.Reschedule(ctx, tenantID, actorUserID, bookingID, newSessionID, reason, canManageTenantBookings(actorRoles))
}

func (s *Service) Cancel(ctx context.Context, tenantID, actorUserID string, actorRoles []rbac.Role, bookingID, reason string) error {
	return s.repo.Cancel(ctx, tenantID, actorUserID, bookingID, reason, 24*time.Hour, canManageTenantBookings(actorRoles))
}

func (s *Service) CheckIn(ctx context.Context, tenantID, actorUserID, bookingID, reason string) error {
	return s.repo.CheckIn(ctx, tenantID, actorUserID, bookingID, reason)
}

func canManageTenantBookings(roles []rbac.Role) bool {
	for _, role := range roles {
		if role == rbac.RoleAdministrator || role == rbac.RoleProgramCoordinator {
			return true
		}
	}
	return false
}
