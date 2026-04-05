package booking

import "time"

type State string

const (
	StateHeld        State = "held"
	StateConfirmed   State = "confirmed"
	StateRescheduled State = "rescheduled"
	StateCanceled    State = "canceled"
	StateCheckedIn   State = "checked_in"
)

type Session struct {
	SessionID        string
	TenantID         string
	RoomID           string
	InstructorUserID *string
	Capacity         int
	StartsAt         time.Time
	EndsAt           time.Time
}

type Booking struct {
	BookingID       string
	TenantID        string
	SessionID       string
	LearnerUserID   string
	State           State
	HoldExpiresAt   *time.Time
	RescheduleCount int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
