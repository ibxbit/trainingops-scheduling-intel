package planning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"trainingops/backend/internal/dbctx"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrVersionConflict    = errors.New("version conflict")
	ErrCircularDependency = errors.New("circular dependency")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreatePlan(ctx context.Context, tenantID, userID, name, description string, startsOn, endsOn *string) (*Plan, error) {
	p := &Plan{}
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO plans (tenant_id, name, description, starts_on, ends_on, created_by_user_id, updated_by_user_id)
VALUES ($1::uuid, $2, $3, NULLIF($4, '')::date, NULLIF($5, '')::date, $6::uuid, $6::uuid)
RETURNING plan_id::text, name, coalesce(description, ''), starts_on::text, ends_on::text, is_active, lock_version, created_at
`, tenantID, name, description, nullableDate(startsOn), nullableDate(endsOn), userID).Scan(
		&p.PlanID,
		&p.Name,
		&p.Description,
		&p.StartsOn,
		&p.EndsOn,
		&p.IsActive,
		&p.LockVersion,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repository) CreateMilestone(ctx context.Context, tenantID, userID, planID, title, description string, dueDate *string, sortOrder int) (*Milestone, error) {
	m := &Milestone{}
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO milestones (tenant_id, plan_id, title, description, due_date, sort_order, created_by_user_id, updated_by_user_id)
VALUES ($1::uuid, $2::uuid, $3, $4, NULLIF($5, '')::date, $6, $7::uuid, $7::uuid)
RETURNING milestone_id::text, plan_id::text, title, coalesce(description, ''), due_date::text, sort_order, lock_version, created_at
`, tenantID, planID, title, description, nullableDate(dueDate), sortOrder, userID).Scan(
		&m.MilestoneID,
		&m.PlanID,
		&m.Title,
		&m.Description,
		&m.DueDate,
		&m.SortOrder,
		&m.LockVersion,
		&m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *Repository) CreateTask(ctx context.Context, tenantID, userID, milestoneID, title, description, state string, dueAt *time.Time, estimatedMinutes, sortOrder int, assigneeUserID *string) (*Task, error) {
	t := &Task{}
	var assignee sql.NullString
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO tasks (
  tenant_id, milestone_id, title, description, state, due_at,
  estimated_minutes, actual_minutes, sort_order, assignee_user_id,
  created_by_user_id, updated_by_user_id
)
VALUES ($1::uuid, $2::uuid, $3, $4, $5::task_state, $6,
        $7, 0, $8, NULLIF($9, '')::uuid,
        $10::uuid, $10::uuid)
RETURNING task_id::text, milestone_id::text, title, coalesce(description, ''), state::text, due_at,
          estimated_minutes, actual_minutes, sort_order, assignee_user_id::text, lock_version, created_at
`, tenantID, milestoneID, title, description, state, dueAt,
		estimatedMinutes, sortOrder, nullableID(assigneeUserID), userID).Scan(
		&t.TaskID,
		&t.MilestoneID,
		&t.Title,
		&t.Description,
		&t.State,
		&t.DueAt,
		&t.EstimatedMinutes,
		&t.ActualMinutes,
		&t.SortOrder,
		&assignee,
		&t.LockVersion,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if assignee.Valid {
		t.AssigneeUserID = &assignee.String
	}
	return t, nil
}

func (r *Repository) AddTaskDependency(ctx context.Context, tenantID, userID, taskID, dependsOnTaskID string) error {
	if taskID == dependsOnTaskID {
		return ErrCircularDependency
	}
	var cyclic bool
	err := dbctx.QueryRowContext(ctx, r.db, `
WITH RECURSIVE deps AS (
  SELECT td.depends_on_task_id
  FROM task_dependencies td
  WHERE td.tenant_id::text = $1 AND td.task_id::text = $2
  UNION ALL
  SELECT td2.depends_on_task_id
  FROM task_dependencies td2
  JOIN deps d ON d.depends_on_task_id = td2.task_id
  WHERE td2.tenant_id::text = $1
)
SELECT EXISTS (SELECT 1 FROM deps WHERE depends_on_task_id::text = $3)
`, tenantID, dependsOnTaskID, taskID).Scan(&cyclic)
	if err != nil {
		return err
	}
	if cyclic {
		return ErrCircularDependency
	}

	_, err = dbctx.ExecContext(ctx, r.db, `
INSERT INTO task_dependencies (tenant_id, task_id, depends_on_task_id, created_by_user_id)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
ON CONFLICT DO NOTHING
`, tenantID, taskID, dependsOnTaskID, userID)
	return err
}

func (r *Repository) RemoveTaskDependency(ctx context.Context, tenantID, taskID, dependsOnTaskID string) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
DELETE FROM task_dependencies
WHERE tenant_id::text = $1 AND task_id::text = $2 AND depends_on_task_id::text = $3
`, tenantID, taskID, dependsOnTaskID)
	return err
}

func (r *Repository) ReorderMilestones(ctx context.Context, tenantID, planID string, orderedMilestoneIDs []string, userID string) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for idx, id := range orderedMilestoneIDs {
		_, err := tx.ExecContext(ctx, `
UPDATE milestones
SET sort_order = $4, updated_by_user_id = $5::uuid, updated_at = NOW(), lock_version = lock_version + 1
WHERE tenant_id::text = $1 AND plan_id::text = $2 AND milestone_id::text = $3
`, tenantID, planID, id, idx, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) ReorderTasks(ctx context.Context, tenantID, milestoneID string, orderedTaskIDs []string, userID string) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for idx, id := range orderedTaskIDs {
		_, err := tx.ExecContext(ctx, `
UPDATE tasks
SET sort_order = $4, updated_by_user_id = $5::uuid, updated_at = NOW(), lock_version = lock_version + 1
WHERE tenant_id::text = $1 AND milestone_id::text = $2 AND task_id::text = $3
`, tenantID, milestoneID, id, idx, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

type BulkTaskPatch struct {
	TaskIDs          []string
	State            *string
	DueAt            *time.Time
	EstimatedMinutes *int
	ActualMinutes    *int
	AssigneeUserID   *string
	MilestoneID      *string
}

func (r *Repository) BulkUpdateTasks(ctx context.Context, tenantID, userID string, patch BulkTaskPatch) error {
	if len(patch.TaskIDs) == 0 {
		return nil
	}

	sets := []string{"updated_by_user_id = $2::uuid", "updated_at = NOW()", "lock_version = lock_version + 1"}
	args := []any{tenantID, userID}
	idx := 3

	if patch.State != nil {
		sets = append(sets, fmt.Sprintf("state = $%d::task_state", idx))
		args = append(args, *patch.State)
		idx++
	}
	if patch.DueAt != nil {
		sets = append(sets, fmt.Sprintf("due_at = $%d", idx))
		args = append(args, *patch.DueAt)
		idx++
	}
	if patch.EstimatedMinutes != nil {
		sets = append(sets, fmt.Sprintf("estimated_minutes = $%d", idx))
		args = append(args, *patch.EstimatedMinutes)
		idx++
	}
	if patch.ActualMinutes != nil {
		sets = append(sets, fmt.Sprintf("actual_minutes = $%d", idx))
		args = append(args, *patch.ActualMinutes)
		idx++
	}
	if patch.AssigneeUserID != nil {
		sets = append(sets, fmt.Sprintf("assignee_user_id = NULLIF($%d, '')::uuid", idx))
		args = append(args, *patch.AssigneeUserID)
		idx++
	}
	if patch.MilestoneID != nil {
		sets = append(sets, fmt.Sprintf("milestone_id = $%d::uuid", idx))
		args = append(args, *patch.MilestoneID)
		idx++
	}

	if len(sets) == 3 {
		return nil
	}

	inPlaceholders := make([]string, 0, len(patch.TaskIDs))
	for _, id := range patch.TaskIDs {
		inPlaceholders = append(inPlaceholders, fmt.Sprintf("$%d::uuid", idx))
		args = append(args, id)
		idx++
	}

	q := fmt.Sprintf(`
UPDATE tasks
SET %s
WHERE tenant_id::text = $1 AND task_id IN (%s)
`, strings.Join(sets, ", "), strings.Join(inPlaceholders, ","))

	_, err := dbctx.ExecContext(ctx, r.db, q, args...)
	return err
}

func (r *Repository) PlanTree(ctx context.Context, tenantID, planID string) (*PlanTree, error) {
	pt := &PlanTree{}
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT plan_id::text, name, coalesce(description, ''), starts_on::text, ends_on::text, is_active, lock_version, created_at
FROM plans
WHERE tenant_id::text = $1 AND plan_id::text = $2
`, tenantID, planID).Scan(
		&pt.Plan.PlanID,
		&pt.Plan.Name,
		&pt.Plan.Description,
		&pt.Plan.StartsOn,
		&pt.Plan.EndsOn,
		&pt.Plan.IsActive,
		&pt.Plan.LockVersion,
		&pt.Plan.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	mRows, err := dbctx.QueryContext(ctx, r.db, `
SELECT milestone_id::text, plan_id::text, title, coalesce(description, ''), due_date::text, sort_order, lock_version, created_at
FROM milestones
WHERE tenant_id::text = $1 AND plan_id::text = $2
ORDER BY sort_order, created_at
`, tenantID, planID)
	if err != nil {
		return nil, err
	}
	defer mRows.Close()

	pt.Milestones = make([]Milestone, 0)
	for mRows.Next() {
		var m Milestone
		if err := mRows.Scan(&m.MilestoneID, &m.PlanID, &m.Title, &m.Description, &m.DueDate, &m.SortOrder, &m.LockVersion, &m.CreatedAt); err != nil {
			return nil, err
		}
		pt.Milestones = append(pt.Milestones, m)
	}
	if err := mRows.Err(); err != nil {
		return nil, err
	}

	tRows, err := dbctx.QueryContext(ctx, r.db, `
SELECT task_id::text, milestone_id::text, title, coalesce(description, ''), state::text, due_at,
       estimated_minutes, actual_minutes, sort_order, assignee_user_id::text, lock_version, created_at
FROM tasks
WHERE tenant_id::text = $1
  AND milestone_id IN (SELECT milestone_id FROM milestones WHERE tenant_id::text = $1 AND plan_id::text = $2)
ORDER BY milestone_id, sort_order, created_at
`, tenantID, planID)
	if err != nil {
		return nil, err
	}
	defer tRows.Close()

	pt.Tasks = make([]Task, 0)
	for tRows.Next() {
		var t Task
		var assignee sql.NullString
		if err := tRows.Scan(&t.TaskID, &t.MilestoneID, &t.Title, &t.Description, &t.State, &t.DueAt, &t.EstimatedMinutes, &t.ActualMinutes, &t.SortOrder, &assignee, &t.LockVersion, &t.CreatedAt); err != nil {
			return nil, err
		}
		if assignee.Valid {
			t.AssigneeUserID = &assignee.String
		}
		pt.Tasks = append(pt.Tasks, t)
	}
	if err := tRows.Err(); err != nil {
		return nil, err
	}

	return pt, nil
}

func nullableDate(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func nullableID(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
