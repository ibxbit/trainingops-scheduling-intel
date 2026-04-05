package dashboard

import (
	"context"
	"errors"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Refresh(ctx context.Context, tenantID, userID, metricDate string) (string, error) {
	if metricDate == "" {
		metricDate = todayDateUTC()
	}
	refreshID, err := s.repo.StartRefresh(ctx, tenantID, userID, metricDate)
	if err != nil {
		return "", err
	}
	if err := s.repo.Precompute(ctx, tenantID, metricDate); err != nil {
		errMsg := err.Error()
		_ = s.repo.FinishRefresh(ctx, tenantID, refreshID, &errMsg)
		return "", err
	}
	_ = s.repo.FinishRefresh(ctx, tenantID, refreshID, nil)
	return refreshID, nil
}

func (s *Service) Overview(ctx context.Context, tenantID, metricDate string) (*Overview, error) {
	if metricDate == "" {
		metricDate = todayDateUTC()
	}
	return s.repo.Overview(ctx, tenantID, metricDate)
}

func (s *Service) RunNightlyFeatureBatch(ctx context.Context, tenantID, userID, featureDate string) ([]string, error) {
	if featureDate == "" {
		featureDate = time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
	}
	windows := []int{7, 30, 90}
	batchIDs := make([]string, 0, len(windows))
	for _, window := range windows {
		batchID, err := s.repo.StartFeatureBatch(ctx, tenantID, userID, featureDate, window)
		if err != nil {
			return nil, err
		}
		if err := s.repo.ComputeFeatureWindow(ctx, tenantID, featureDate, window); err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishFeatureBatch(ctx, tenantID, batchID, &errMsg)
			return nil, err
		}
		_ = s.repo.FinishFeatureBatch(ctx, tenantID, batchID, nil)
		batchIDs = append(batchIDs, batchID)
	}
	return batchIDs, nil
}

func (s *Service) LearnerFeatures(ctx context.Context, tenantID, featureDate string, windowDays, limit int, segment string) ([]LearnerFeature, error) {
	if featureDate == "" {
		featureDate = todayDateUTC()
	}
	if !validFeatureWindow(windowDays) {
		return nil, errors.New("window_days must be 7, 30, or 90")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.LearnerFeatures(ctx, tenantID, featureDate, windowDays, limit, segment)
}

func (s *Service) CohortFeatures(ctx context.Context, tenantID, featureDate string, windowDays, limit int) ([]CohortFeature, error) {
	if featureDate == "" {
		featureDate = todayDateUTC()
	}
	if !validFeatureWindow(windowDays) {
		return nil, errors.New("window_days must be 7, 30, or 90")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.CohortFeatures(ctx, tenantID, featureDate, windowDays, limit)
}

func (s *Service) ReportingMetrics(ctx context.Context, tenantID, featureDate string, windowDays int) ([]ReportingMetric, error) {
	if featureDate == "" {
		featureDate = todayDateUTC()
	}
	if !validFeatureWindow(windowDays) {
		return nil, errors.New("window_days must be 7, 30, or 90")
	}
	return s.repo.ReportingMetrics(ctx, tenantID, featureDate, windowDays)
}

func (s *Service) TodaySessions(ctx context.Context, tenantID, metricDate string, limit int) ([]TodaySession, error) {
	if metricDate == "" {
		metricDate = todayDateUTC()
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.TodaySessions(ctx, tenantID, metricDate, limit)
}

func validFeatureWindow(windowDays int) bool {
	return windowDays == 7 || windowDays == 30 || windowDays == 90
}
