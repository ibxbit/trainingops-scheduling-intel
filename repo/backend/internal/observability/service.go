package observability

import (
	"context"
	"errors"
	"path/filepath"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ApplyRetention(ctx context.Context, days int) error {
	if days <= 0 {
		days = 90
	}
	return s.repo.ApplyRetention(ctx, days)
}

func (s *Service) LogWorkflow(ctx context.Context, tenantID string, userID *string, workflowName, resourceID, outcome string, statusCode, latencyMS int, details map[string]any) {
	_ = s.repo.InsertWorkflowLog(ctx, tenantID, userID, workflowName, resourceID, outcome, statusCode, latencyMS, details)
}

func (s *Service) WorkflowLogs(ctx context.Context, tenantID string, limit int) ([]WorkflowLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListWorkflowLogs(ctx, tenantID, limit)
}

func (s *Service) RecordScrapingError(ctx context.Context, tenantID, sourceName, errorCode, errorMessage string, metadata map[string]any) error {
	if sourceName == "" || errorMessage == "" {
		return errors.New("source_name and error_message are required")
	}
	return s.repo.InsertScrapingError(ctx, tenantID, sourceName, errorCode, errorMessage, metadata)
}

func (s *Service) DetectAnomalies(ctx context.Context, tenantID, date string) (int, error) {
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}
	return s.repo.DetectAnomalies(ctx, tenantID, date)
}

func (s *Service) ListAnomalies(ctx context.Context, tenantID, date string, limit int) ([]AnomalyEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListAnomalies(ctx, tenantID, date, limit)
}

func (s *Service) CreateSchedule(ctx context.Context, tenantID, userID, name, format, frequency, folder string, nextRunAt *time.Time) (*ReportSchedule, error) {
	if name == "" || folder == "" {
		return nil, errors.New("name and output_folder are required")
	}
	if format != "csv" && format != "pdf" {
		return nil, errors.New("format must be csv or pdf")
	}
	if frequency != "daily" && frequency != "weekly" {
		return nil, errors.New("frequency must be daily or weekly")
	}
	runAt := time.Now().UTC()
	if nextRunAt != nil {
		runAt = nextRunAt.UTC()
	}
	return s.repo.CreateSchedule(ctx, tenantID, userID, name, format, frequency, folder, runAt)
}

func (s *Service) RunDueSchedules(ctx context.Context, tenantID string) ([]ReportExport, error) {
	now := time.Now().UTC()
	schedules, err := s.repo.DueSchedules(ctx, tenantID, now)
	if err != nil {
		return nil, err
	}
	exports := make([]ReportExport, 0, len(schedules))
	for _, sch := range schedules {
		exp, err := s.runOne(ctx, tenantID, &sch, now.Format("2006-01-02"))
		if err == nil && exp != nil {
			exports = append(exports, *exp)
		}
		_ = s.repo.AdvanceScheduleNextRun(ctx, tenantID, sch.ScheduleID, sch.Frequency, now)
	}
	return exports, nil
}

func (s *Service) RunScheduleNow(ctx context.Context, tenantID, scheduleID, reportDate string) (*ReportExport, error) {
	if reportDate == "" {
		reportDate = time.Now().UTC().Format("2006-01-02")
	}
	sch, err := s.repo.ScheduleByID(ctx, tenantID, scheduleID)
	if err != nil {
		return nil, err
	}
	return s.runOne(ctx, tenantID, sch, reportDate)
}

func (s *Service) ListExports(ctx context.Context, tenantID string, limit int) ([]ReportExport, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListExports(ctx, tenantID, limit)
}

func (s *Service) runOne(ctx context.Context, tenantID string, sch *ReportSchedule, reportDate string) (*ReportExport, error) {
	summary, kpis, heatmap, err := s.repo.DashboardDataForReport(ctx, tenantID, reportDate)
	if err != nil {
		errMsg := err.Error()
		return s.repo.InsertExport(ctx, tenantID, &sch.ScheduleID, reportDate, sch.Format, "failed", nil, nil, &errMsg)
	}

	ext := ".csv"
	if sch.Format == "pdf" {
		ext = ".pdf"
	}
	name := sanitizeFileName(sch.Name)
	fullPath := filepath.Join(sch.OutputFolder, name+"_"+reportDate+ext)

	var size int64
	if sch.Format == "pdf" {
		size, err = writePDF(fullPath, summary, kpis, heatmap)
	} else {
		size, err = writeCSV(fullPath, summary, kpis, heatmap)
	}
	if err != nil {
		errMsg := err.Error()
		return s.repo.InsertExport(ctx, tenantID, &sch.ScheduleID, reportDate, sch.Format, "failed", nil, nil, &errMsg)
	}

	return s.repo.InsertExport(ctx, tenantID, &sch.ScheduleID, reportDate, sch.Format, "success", &fullPath, &size, nil)
}

func sanitizeFileName(v string) string {
	b := make([]rune, 0, len(v))
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b = append(b, r)
		} else if r == ' ' {
			b = append(b, '_')
		}
	}
	if len(b) == 0 {
		return "report"
	}
	return string(b)
}
