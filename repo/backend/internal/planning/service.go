package planning

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

func (s *Service) CreatePlan(ctx context.Context, tenantID, userID, name, description string, startsOn, endsOn *string) (*Plan, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	return s.repo.CreatePlan(ctx, tenantID, userID, name, description, startsOn, endsOn)
}

func (s *Service) CreateMilestone(ctx context.Context, tenantID, userID, planID, title, description string, dueDate *string, sortOrder int) (*Milestone, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	return s.repo.CreateMilestone(ctx, tenantID, userID, planID, title, description, dueDate, sortOrder)
}

func (s *Service) CreateTask(ctx context.Context, tenantID, userID, milestoneID, title, description, state string, dueAt *time.Time, estimatedMinutes, sortOrder int, assigneeUserID *string) (*Task, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	if state == "" {
		state = "todo"
	}
	if estimatedMinutes < 0 {
		return nil, errors.New("estimated_minutes must be >= 0")
	}
	return s.repo.CreateTask(ctx, tenantID, userID, milestoneID, title, description, state, dueAt, estimatedMinutes, sortOrder, assigneeUserID)
}

func (s *Service) AddTaskDependency(ctx context.Context, tenantID, userID, taskID, dependsOnTaskID string) error {
	return s.repo.AddTaskDependency(ctx, tenantID, userID, taskID, dependsOnTaskID)
}

func (s *Service) RemoveTaskDependency(ctx context.Context, tenantID, taskID, dependsOnTaskID string) error {
	return s.repo.RemoveTaskDependency(ctx, tenantID, taskID, dependsOnTaskID)
}

func (s *Service) ReorderMilestones(ctx context.Context, tenantID, userID, planID string, orderedMilestoneIDs []string) error {
	return s.repo.ReorderMilestones(ctx, tenantID, planID, orderedMilestoneIDs, userID)
}

func (s *Service) ReorderTasks(ctx context.Context, tenantID, userID, milestoneID string, orderedTaskIDs []string) error {
	return s.repo.ReorderTasks(ctx, tenantID, milestoneID, orderedTaskIDs, userID)
}

func (s *Service) BulkUpdateTasks(ctx context.Context, tenantID, userID string, patch BulkTaskPatch) error {
	if patch.EstimatedMinutes != nil && *patch.EstimatedMinutes < 0 {
		return errors.New("estimated_minutes must be >= 0")
	}
	if patch.ActualMinutes != nil && *patch.ActualMinutes < 0 {
		return errors.New("actual_minutes must be >= 0")
	}
	return s.repo.BulkUpdateTasks(ctx, tenantID, userID, patch)
}

func (s *Service) PlanTree(ctx context.Context, tenantID, planID string) (*PlanTree, error) {
	return s.repo.PlanTree(ctx, tenantID, planID)
}
