package planning

import "time"

type Plan struct {
	PlanID      string    `json:"plan_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartsOn    *string   `json:"starts_on"`
	EndsOn      *string   `json:"ends_on"`
	IsActive    bool      `json:"is_active"`
	LockVersion int       `json:"lock_version"`
	CreatedAt   time.Time `json:"created_at"`
}

type Milestone struct {
	MilestoneID string    `json:"milestone_id"`
	PlanID      string    `json:"plan_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     *string   `json:"due_date"`
	SortOrder   int       `json:"sort_order"`
	LockVersion int       `json:"lock_version"`
	CreatedAt   time.Time `json:"created_at"`
}

type Task struct {
	TaskID           string     `json:"task_id"`
	MilestoneID      string     `json:"milestone_id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	State            string     `json:"state"`
	DueAt            *time.Time `json:"due_at"`
	EstimatedMinutes int        `json:"estimated_minutes"`
	ActualMinutes    int        `json:"actual_minutes"`
	SortOrder        int        `json:"sort_order"`
	LockVersion      int        `json:"lock_version"`
	AssigneeUserID   *string    `json:"assignee_user_id"`
	CreatedAt        time.Time  `json:"created_at"`
}

type PlanTree struct {
	Plan       Plan        `json:"plan"`
	Milestones []Milestone `json:"milestones"`
	Tasks      []Task      `json:"tasks"`
}
