package content

import "time"

type Document struct {
	DocumentID       string `json:"document_id"`
	TenantID         string `json:"tenant_id"`
	Title            string `json:"title"`
	Summary          string `json:"summary"`
	Difficulty       int    `json:"difficulty"`
	DurationMinutes  int    `json:"duration_minutes"`
	CurrentVersionNo int    `json:"current_version_no"`
	IsArchived       bool   `json:"is_archived"`
}

type DocumentVersion struct {
	DocumentVersionID string    `json:"document_version_id"`
	DocumentID        string    `json:"document_id"`
	VersionNo         int       `json:"version_no"`
	FileName          string    `json:"file_name"`
	StoragePath       string    `json:"storage_path"`
	MimeType          string    `json:"mime_type"`
	FileSizeBytes     int64     `json:"file_size_bytes"`
	SHA256Checksum    string    `json:"sha256_checksum"`
	CreatedAt         time.Time `json:"created_at"`
}

type UploadSession struct {
	UploadID       string     `json:"upload_id"`
	DocumentID     *string    `json:"document_id"`
	FileName       string     `json:"file_name"`
	MimeType       string     `json:"mime_type"`
	TotalChunks    int        `json:"total_chunks"`
	ChunkSizeBytes int        `json:"chunk_size_bytes"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CompletedAt    *time.Time `json:"completed_at"`
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
