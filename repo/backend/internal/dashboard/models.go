package dashboard

import "time"

type DailySummary struct {
	MetricDate       string `json:"metric_date"`
	TodaysSessions   int    `json:"todays_sessions"`
	PendingApprovals int    `json:"pending_approvals"`
}

type KPI struct {
	MetricKey   string  `json:"metric_key"`
	MetricValue float64 `json:"metric_value"`
	Numerator   float64 `json:"numerator"`
	Denominator float64 `json:"denominator"`
}

type HeatmapCell struct {
	HourBucket    int     `json:"hour_bucket"`
	RoomID        *string `json:"room_id"`
	SessionsCount int     `json:"sessions_count"`
	BookedSeats   int     `json:"booked_seats"`
	TotalSeats    int     `json:"total_seats"`
	OccupancyRate float64 `json:"occupancy_rate"`
}

type Overview struct {
	Summary DailySummary  `json:"summary"`
	KPIs    []KPI         `json:"kpis"`
	Heatmap []HeatmapCell `json:"heatmap"`
}

type RefreshRun struct {
	RefreshID  string     `json:"refresh_id"`
	MetricDate string     `json:"metric_date"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

type TodaySession struct {
	SessionID      string    `json:"session_id"`
	Title          string    `json:"title"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	RoomID         string    `json:"room_id"`
	Capacity       int       `json:"capacity"`
	BookedSeats    int       `json:"booked_seats"`
	OccupancyRate  float64   `json:"occupancy_rate"`
	InstructorUser *string   `json:"instructor_user_id"`
}

type LearnerFeature struct {
	FeatureDate      string  `json:"feature_date"`
	WindowDays       int     `json:"window_days"`
	LearnerUserID    string  `json:"learner_user_id"`
	SessionsBooked   int     `json:"sessions_booked"`
	SessionsAttended int     `json:"sessions_attended"`
	AttendanceRate   float64 `json:"attendance_rate"`
	ActiveDays       int     `json:"active_days"`
	StudyMinutes     int     `json:"study_minutes"`
	ContentPreviews  int     `json:"content_previews"`
	ContentDownloads int     `json:"content_downloads"`
	CommunityEvents  int     `json:"community_events"`
	EngagementScore  float64 `json:"engagement_score"`
	Segment          string  `json:"segment"`
}

type CohortFeature struct {
	FeatureDate         string         `json:"feature_date"`
	WindowDays          int            `json:"window_days"`
	CohortID            string         `json:"cohort_id"`
	MembersCount        int            `json:"members_count"`
	ActiveLearners      int            `json:"active_learners"`
	AvgAttendanceRate   float64        `json:"avg_attendance_rate"`
	AvgStudyMinutes     float64        `json:"avg_study_minutes"`
	AvgEngagementScore  float64        `json:"avg_engagement_score"`
	SegmentDistribution map[string]int `json:"segment_distribution"`
}

type ReportingMetric struct {
	FeatureDate string   `json:"feature_date"`
	WindowDays  int      `json:"window_days"`
	MetricKey   string   `json:"metric_key"`
	MetricValue float64  `json:"metric_value"`
	Numerator   *float64 `json:"numerator"`
	Denominator *float64 `json:"denominator"`
}
