package calendar

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CheckAvailability(ctx context.Context, tenantID, sessionID string) (AvailabilityReason, []Alternative, error) {
	reason, err := s.repo.CheckAvailability(ctx, tenantID, sessionID)
	if err != nil {
		return "", nil, err
	}
	if reason == ReasonAvailable {
		return reason, nil, nil
	}
	alts, err := s.repo.SuggestAlternatives(ctx, tenantID, sessionID, 3)
	if err != nil {
		return "", nil, err
	}
	return reason, alts, nil
}

func (s *Service) CreateTimeSlotRule(ctx context.Context, tenantID string, in TimeSlotRule) (string, error) {
	return s.repo.CreateTimeSlotRule(ctx, tenantID, in)
}

func (s *Service) UpdateTimeSlotRule(ctx context.Context, tenantID, ruleID string, in TimeSlotRule) error {
	return s.repo.UpdateTimeSlotRule(ctx, tenantID, ruleID, in)
}

func (s *Service) CreateBlackoutDate(ctx context.Context, tenantID string, in BlackoutDate) (string, error) {
	return s.repo.CreateBlackoutDate(ctx, tenantID, in)
}

func (s *Service) UpdateBlackoutDate(ctx context.Context, tenantID, blackoutID string, in BlackoutDate) error {
	return s.repo.UpdateBlackoutDate(ctx, tenantID, blackoutID, in)
}

func (s *Service) CreateAcademicTerm(ctx context.Context, tenantID string, in AcademicTerm) (string, error) {
	return s.repo.CreateAcademicTerm(ctx, tenantID, in)
}

func (s *Service) UpdateAcademicTerm(ctx context.Context, tenantID, termID string, in AcademicTerm) error {
	return s.repo.UpdateAcademicTerm(ctx, tenantID, termID, in)
}
