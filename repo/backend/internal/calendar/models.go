package calendar

import "time"

type AvailabilityReason string

const (
	ReasonAvailable               AvailabilityReason = "available"
	ReasonOutsideAcademicCalendar AvailabilityReason = "outside_academic_calendar"
	ReasonBlackoutDate            AvailabilityReason = "blackout_date"
	ReasonOutsideAllowedSlot      AvailabilityReason = "outside_allowed_slot"
	ReasonRoomOccupied            AvailabilityReason = "room_occupied"
	ReasonInstructorUnavailable   AvailabilityReason = "instructor_unavailable"
	ReasonCapacityReached         AvailabilityReason = "capacity_reached"
)

type Alternative struct {
	SessionID string    `json:"session_id"`
	RoomID    string    `json:"room_id"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
}

type TimeSlotRule struct {
	RuleID      string
	TenantID    string
	RoomID      *string
	Weekday     int
	SlotStart   string
	SlotEnd     string
	IsActive    bool
	LockVersion int
}

type BlackoutDate struct {
	BlackoutID   string
	TenantID     string
	RoomID       *string
	BlackoutDate string
	Reason       string
	IsActive     bool
	LockVersion  int
}

type AcademicTerm struct {
	TermID      string
	TenantID    string
	Name        string
	StartDate   string
	EndDate     string
	IsActive    bool
	LockVersion int
}
