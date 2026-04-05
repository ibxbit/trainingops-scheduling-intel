package planning

import (
	"errors"
	"net/http"
	"time"

	"trainingops/backend/internal/access"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createPlanRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	StartsOn    *string `json:"starts_on"`
	EndsOn      *string `json:"ends_on"`
}

func (h *Handler) CreatePlan(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req createPlanRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	p, err := h.svc.CreatePlan(c.Request().Context(), tenantID, userID, req.Name, req.Description, req.StartsOn, req.EndsOn)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": p})
}

type createMilestoneRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	DueDate     *string `json:"due_date"`
	SortOrder   int     `json:"sort_order"`
}

func (h *Handler) CreateMilestone(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	planID := c.Param("plan_id")
	var req createMilestoneRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	m, err := h.svc.CreateMilestone(c.Request().Context(), tenantID, userID, planID, req.Title, req.Description, req.DueDate, req.SortOrder)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": m})
}

type createTaskRequest struct {
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	State            string  `json:"state"`
	DueAt            *string `json:"due_at"`
	EstimatedMinutes int     `json:"estimated_minutes"`
	SortOrder        int     `json:"sort_order"`
	AssigneeUserID   *string `json:"assignee_user_id"`
}

func (h *Handler) CreateTask(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	milestoneID := c.Param("milestone_id")
	var req createTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	dueAt, err := parseTime(req.DueAt)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid due_at"})
	}
	t, err := h.svc.CreateTask(c.Request().Context(), tenantID, userID, milestoneID, req.Title, req.Description, req.State, dueAt, req.EstimatedMinutes, req.SortOrder, req.AssigneeUserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"data": t})
}

type dependencyRequest struct {
	DependsOnTaskID string `json:"depends_on_task_id"`
}

func (h *Handler) AddDependency(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	taskID := c.Param("task_id")
	var req dependencyRequest
	if err := c.Bind(&req); err != nil || req.DependsOnTaskID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "depends_on_task_id is required"})
	}
	err := h.svc.AddTaskDependency(c.Request().Context(), tenantID, userID, taskID, req.DependsOnTaskID)
	if err != nil {
		if errors.Is(err, ErrCircularDependency) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "circular dependency"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "added"}})
}

func (h *Handler) RemoveDependency(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	taskID := c.Param("task_id")
	dependsOnTaskID := c.Param("depends_on_task_id")
	if err := h.svc.RemoveTaskDependency(c.Request().Context(), tenantID, taskID, dependsOnTaskID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "removed"}})
}

type reorderRequest struct {
	OrderedIDs []string `json:"ordered_ids"`
}

func (h *Handler) ReorderMilestones(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	planID := c.Param("plan_id")
	var req reorderRequest
	if err := c.Bind(&req); err != nil || len(req.OrderedIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ordered_ids required"})
	}
	if err := h.svc.ReorderMilestones(c.Request().Context(), tenantID, userID, planID, req.OrderedIDs); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "reordered"}})
}

func (h *Handler) ReorderTasks(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	milestoneID := c.Param("milestone_id")
	var req reorderRequest
	if err := c.Bind(&req); err != nil || len(req.OrderedIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ordered_ids required"})
	}
	if err := h.svc.ReorderTasks(c.Request().Context(), tenantID, userID, milestoneID, req.OrderedIDs); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "reordered"}})
}

type bulkUpdateTasksRequest struct {
	TaskIDs          []string `json:"task_ids"`
	State            *string  `json:"state"`
	DueAt            *string  `json:"due_at"`
	EstimatedMinutes *int     `json:"estimated_minutes"`
	ActualMinutes    *int     `json:"actual_minutes"`
	AssigneeUserID   *string  `json:"assignee_user_id"`
	MilestoneID      *string  `json:"milestone_id"`
}

func (h *Handler) BulkUpdateTasks(c echo.Context) error {
	tenantID, userID, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	var req bulkUpdateTasksRequest
	if err := c.Bind(&req); err != nil || len(req.TaskIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "task_ids required"})
	}

	dueAt, err := parseTime(req.DueAt)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid due_at"})
	}

	err = h.svc.BulkUpdateTasks(c.Request().Context(), tenantID, userID, BulkTaskPatch{
		TaskIDs:          req.TaskIDs,
		State:            req.State,
		DueAt:            dueAt,
		EstimatedMinutes: req.EstimatedMinutes,
		ActualMinutes:    req.ActualMinutes,
		AssigneeUserID:   req.AssigneeUserID,
		MilestoneID:      req.MilestoneID,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": map[string]string{"status": "updated"}})
}

func (h *Handler) PlanTree(c echo.Context) error {
	tenantID, _, ok := identity(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	planID := c.Param("plan_id")
	out, err := h.svc.PlanTree(c.Request().Context(), tenantID, planID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "plan not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "load failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

func identity(c echo.Context) (string, string, bool) {
	tenantID, okTenant := c.Get(access.ContextTenantID).(string)
	userID, okUser := c.Get(access.ContextUserID).(string)
	if !okTenant || !okUser || tenantID == "" || userID == "" {
		return "", "", false
	}
	return tenantID, userID, true
}

func parseTime(v *string) (*time.Time, error) {
	if v == nil || *v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *v)
	if err != nil {
		return nil, err
	}
	t = t.UTC()
	return &t, nil
}
