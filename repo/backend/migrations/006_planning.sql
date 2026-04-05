BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'task_state') THEN
    CREATE TYPE task_state AS ENUM (
      'todo',
      'in_progress',
      'blocked',
      'done',
      'canceled'
    );
  END IF;
END$$;

CREATE TABLE IF NOT EXISTS plans (
  plan_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  name                 TEXT NOT NULL,
  description          TEXT,
  starts_on            DATE,
  ends_on              DATE,
  is_active            BOOLEAN NOT NULL DEFAULT TRUE,
  lock_version         INTEGER NOT NULL DEFAULT 0,
  created_by_user_id   UUID NOT NULL,
  updated_by_user_id   UUID,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (ends_on IS NULL OR starts_on IS NULL OR ends_on >= starts_on),
  UNIQUE (tenant_id, plan_id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS milestones (
  milestone_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  plan_id              UUID NOT NULL,
  title                TEXT NOT NULL,
  description          TEXT,
  due_date             DATE,
  sort_order           INTEGER NOT NULL DEFAULT 0,
  lock_version         INTEGER NOT NULL DEFAULT 0,
  created_by_user_id   UUID NOT NULL,
  updated_by_user_id   UUID,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, milestone_id),
  FOREIGN KEY (tenant_id, plan_id) REFERENCES plans(tenant_id, plan_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_milestones_tenant_plan_order
  ON milestones (tenant_id, plan_id, sort_order, created_at);

CREATE TABLE IF NOT EXISTS tasks (
  task_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  milestone_id          UUID NOT NULL,
  title                 TEXT NOT NULL,
  description           TEXT,
  state                 task_state NOT NULL DEFAULT 'todo',
  due_at                TIMESTAMPTZ,
  estimated_minutes     INTEGER NOT NULL DEFAULT 0 CHECK (estimated_minutes >= 0),
  actual_minutes        INTEGER NOT NULL DEFAULT 0 CHECK (actual_minutes >= 0),
  sort_order            INTEGER NOT NULL DEFAULT 0,
  lock_version          INTEGER NOT NULL DEFAULT 0,
  assignee_user_id      UUID,
  created_by_user_id    UUID NOT NULL,
  updated_by_user_id    UUID,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, task_id),
  FOREIGN KEY (tenant_id, milestone_id) REFERENCES milestones(tenant_id, milestone_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, assignee_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL,
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_tenant_milestone_order
  ON tasks (tenant_id, milestone_id, sort_order, created_at);

CREATE INDEX IF NOT EXISTS idx_tasks_tenant_due
  ON tasks (tenant_id, due_at, state);

CREATE TABLE IF NOT EXISTS task_dependencies (
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  task_id               UUID NOT NULL,
  depends_on_task_id    UUID NOT NULL,
  created_by_user_id    UUID,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, task_id, depends_on_task_id),
  FOREIGN KEY (tenant_id, task_id) REFERENCES tasks(tenant_id, task_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, depends_on_task_id) REFERENCES tasks(tenant_id, task_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL,
  CHECK (task_id <> depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_deps_reverse
  ON task_dependencies (tenant_id, depends_on_task_id);

ALTER TABLE plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE milestones ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_dependencies ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_plans ON plans;
CREATE POLICY tenant_isolation_plans ON plans
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_milestones ON milestones;
CREATE POLICY tenant_isolation_milestones ON milestones
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_tasks ON tasks;
CREATE POLICY tenant_isolation_tasks ON tasks
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_task_dependencies ON task_dependencies;
CREATE POLICY tenant_isolation_task_dependencies ON task_dependencies
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
