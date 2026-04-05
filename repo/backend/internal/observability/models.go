package observability

import "time"

type WorkflowLog struct {
	WorkflowLogID string    `json:"workflow_log_id"`
	WorkflowName  string    `json:"workflow_name"`
	ResourceID    string    `json:"resource_id"`
	Outcome       string    `json:"outcome"`
	StatusCode    int       `json:"status_code"`
	LatencyMS     int       `json:"latency_ms"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type ScrapingError struct {
	ScrapingErrorID string    `json:"scraping_error_id"`
	SourceName      string    `json:"source_name"`
	ErrorCode       string    `json:"error_code"`
	ErrorMessage    string    `json:"error_message"`
	OccurredAt      time.Time `json:"occurred_at"`
}

type AnomalyEvent struct {
	AnomalyEventID string    `json:"anomaly_event_id"`
	AnomalyDate    string    `json:"anomaly_date"`
	AnomalyType    string    `json:"anomaly_type"`
	Severity       string    `json:"severity"`
	ObservedValue  float64   `json:"observed_value"`
	BaselineValue  float64   `json:"baseline_value"`
	ThresholdValue float64   `json:"threshold_value"`
	CreatedAt      time.Time `json:"created_at"`
}

type ReportSchedule struct {
	ScheduleID   string    `json:"schedule_id"`
	Name         string    `json:"name"`
	Format       string    `json:"format"`
	Frequency    string    `json:"frequency"`
	OutputFolder string    `json:"output_folder"`
	IsActive     bool      `json:"is_active"`
	NextRunAt    time.Time `json:"next_run_at"`
}

type ReportExport struct {
	ExportID      string    `json:"export_id"`
	ScheduleID    *string   `json:"schedule_id"`
	ReportDate    string    `json:"report_date"`
	Format        string    `json:"format"`
	FilePath      *string   `json:"file_path"`
	FileSizeBytes *int64    `json:"file_size_bytes"`
	Status        string    `json:"status"`
	ErrorMessage  *string   `json:"error_message"`
	CreatedAt     time.Time `json:"created_at"`
}
