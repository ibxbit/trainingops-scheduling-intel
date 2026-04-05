package content

import "time"

type Document struct {
	DocumentID       string
	TenantID         string
	Title            string
	Summary          string
	Difficulty       int
	DurationMinutes  int
	CurrentVersionNo int
	IsArchived       bool
}

type DocumentVersion struct {
	DocumentVersionID string
	DocumentID        string
	VersionNo         int
	FileName          string
	StoragePath       string
	MimeType          string
	FileSizeBytes     int64
	SHA256Checksum    string
	CreatedAt         time.Time
}

type UploadSession struct {
	UploadID       string
	DocumentID     *string
	FileName       string
	MimeType       string
	TotalChunks    int
	ChunkSizeBytes int
	ExpiresAt      time.Time
	CompletedAt    *time.Time
}

type AlternativeDoc struct {
	DocumentID string    `json:"document_id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
}

type IngestionSource struct {
	SourceID                string     `json:"source_id"`
	Name                    string     `json:"name"`
	BaseURL                 string     `json:"base_url"`
	IsActive                bool       `json:"is_active"`
	PausedForManualReview   bool       `json:"paused_for_manual_review"`
	ManualReviewReason      *string    `json:"manual_review_reason"`
	ScheduleIntervalMinutes int        `json:"schedule_interval_minutes"`
	ScheduleJitterSeconds   int        `json:"schedule_jitter_seconds"`
	RateLimitPerMinute      int        `json:"rate_limit_per_minute"`
	RequestTimeoutSeconds   int        `json:"request_timeout_seconds"`
	NextRunAt               time.Time  `json:"next_run_at"`
	LastRunAt               *time.Time `json:"last_run_at"`
	CreatedByUserID         string     `json:"created_by_user_id"`
}

type IngestionRun struct {
	RunID            string     `json:"run_id"`
	SourceID         string     `json:"source_id"`
	TriggerType      string     `json:"trigger_type"`
	Status           string     `json:"status"`
	ProxyURL         *string    `json:"proxy_url"`
	UserAgent        *string    `json:"user_agent"`
	HTTPStatus       *int       `json:"http_status"`
	ResponseBytes    *int64     `json:"response_bytes"`
	RecordsProcessed int        `json:"records_processed"`
	ErrorMessage     *string    `json:"error_message"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	NextRunAt        *time.Time `json:"next_run_at"`
}

type IngestedRecord struct {
	ExternalID   string
	Title        string
	Summary      string
	Category     string
	Tags         []string
	Difficulty   int
	DurationMins int
	BodyText     string
	Metadata     map[string]any
	Checksum     string
}
