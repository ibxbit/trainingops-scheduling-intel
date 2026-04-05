import { useMemo, useState } from "react";

import {
  addDependency,
  bulkUpdateTasks,
  createMilestone,
  createPlan,
  createTask,
  getPlanTree,
  reorderTasks,
  type PlanTree,
} from "../../api/endpoints";
import { AccessGate } from "../../auth/access-control";
import { useSessionStore } from "../../state/session-store";

export function PlanningPage() {
  const role = useSessionStore((s) => s.user?.primaryRole ?? null);
  const [planID, setPlanID] = useState("");
  const [tree, setTree] = useState<PlanTree | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const [planName, setPlanName] = useState("New Plan");
  const [milestoneID, setMilestoneID] = useState("");
  const [taskID, setTaskID] = useState("");
  const [dependsOnTaskID, setDependsOnTaskID] = useState("");
  const [orderedTaskIDs, setOrderedTaskIDs] = useState("");
  const [bulkTaskIDs, setBulkTaskIDs] = useState("");
  const [draggedTaskID, setDraggedTaskID] = useState<string | null>(null);

  const tasksByMilestone = useMemo(() => {
    const map = new Map<string, PlanTree["tasks"]>();
    if (!tree) {
      return map;
    }
    for (const task of tree.tasks) {
      const items = map.get(task.milestone_id) ?? [];
      items.push(task);
      map.set(task.milestone_id, items);
    }
    for (const [milestone, items] of map.entries()) {
      map.set(
        milestone,
        [...items].sort((a, b) => a.sort_order - b.sort_order),
      );
    }
    return map;
  }, [tree]);

  const applyTaskOrder = async (
    sourceMilestoneID: string,
    targetMilestoneID: string,
    targetTaskID: string,
  ) => {
    if (!tree || !draggedTaskID) {
      return;
    }
    if (sourceMilestoneID !== targetMilestoneID) {
      setError("Drag and drop only supports reordering within one milestone");
      return;
    }
    const current = tasksByMilestone.get(targetMilestoneID) ?? [];
    const ordered = reorderByDrop(current, draggedTaskID, targetTaskID);
    if (ordered.length === 0) {
      return;
    }
    const orderedIDs = ordered.map((task) => task.task_id);

    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await reorderTasks(targetMilestoneID, orderedIDs);
      setTree((prev) => {
        if (!prev) return prev;
        const orderMap = new Map(orderedIDs.map((id, idx) => [id, idx]));
        const updatedTasks = prev.tasks.map((task) => {
          if (task.milestone_id !== targetMilestoneID) {
            return task;
          }
          const next = orderMap.get(task.task_id);
          if (next === undefined) {
            return task;
          }
          return { ...task, sort_order: next };
        });
        return { ...prev, tasks: updatedTasks };
      });
      setStatus("Task order updated");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
      setDraggedTaskID(null);
    }
  };

  const loadPlanTree = async () => {
    if (!planID.trim()) {
      setError("Plan ID is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const out = await getPlanTree(planID.trim());
      setTree(out);
      setStatus("Plan tree loaded");
    } catch (e) {
      setError(messageFromError(e));
      setTree(null);
    } finally {
      setLoading(false);
    }
  };

  const onCreatePlan = async () => {
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const out = await createPlan({
        name: planName,
        description: "Created from frontend",
      });
      setPlanID(out.plan_id);
      setStatus(`Plan created: ${out.plan_id}`);
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const onCreateMilestone = async () => {
    if (!planID.trim()) {
      setError("Plan ID is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const out = await createMilestone(planID.trim(), {
        title: "Milestone",
        description: "Created from frontend",
        sort_order: 1,
      });
      setMilestoneID(out.milestone_id);
      setStatus(`Milestone created: ${out.milestone_id}`);
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const onCreateTask = async () => {
    if (!milestoneID.trim()) {
      setError("Milestone ID is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const out = await createTask(milestoneID.trim(), {
        title: "Task",
        description: "Created from frontend",
        state: "todo",
        estimated_minutes: 30,
        sort_order: 1,
      });
      setTaskID(out.task_id);
      setStatus(`Task created: ${out.task_id}`);
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const onAddDependency = async () => {
    if (!taskID.trim() || !dependsOnTaskID.trim()) {
      setError("Both task and dependency IDs are required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await addDependency(taskID.trim(), dependsOnTaskID.trim());
      setStatus("Dependency added");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const onReorderTasks = async () => {
    if (!milestoneID.trim() || !orderedTaskIDs.trim()) {
      setError("Milestone and ordered task ids are required");
      return;
    }
    const ordered = orderedTaskIDs
      .split(",")
      .map((v) => v.trim())
      .filter(Boolean);
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await reorderTasks(milestoneID.trim(), ordered);
      setStatus("Tasks reordered");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const onBulkUpdate = async () => {
    const ids = bulkTaskIDs
      .split(",")
      .map((v) => v.trim())
      .filter(Boolean);
    if (ids.length === 0) {
      setError("Task IDs are required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await bulkUpdateTasks({ task_ids: ids, state: "done" });
      setStatus(`Bulk updated ${ids.length} task(s)`);
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <section>
      <h2>Tasks & Planning</h2>
      {error ? <p className="error">{error}</p> : null}
      {status ? <p>{status}</p> : null}

      <div className="login-panel">
        <h3>Plan Tree</h3>
        <div className="login-row">
          <input
            value={planID}
            onChange={(e) => setPlanID(e.target.value)}
            placeholder="plan id"
          />
          <button onClick={loadPlanTree} disabled={loading || !planID.trim()}>
            {loading ? "Loading..." : "Load Tree"}
          </button>
        </div>
        {!tree ? (
          <p>No plan tree loaded.</p>
        ) : (
          <div>
            <p>
              Plan: {tree.plan.name} ({tree.plan.plan_id})
            </p>
            {tree.milestones.map((m) => (
              <div key={m.milestone_id}>
                <strong>{m.title}</strong>
                <ul>
                  {(tasksByMilestone.get(m.milestone_id) ?? []).map((t) => (
                    <li
                      key={t.task_id}
                      draggable
                      onDragStart={() => setDraggedTaskID(t.task_id)}
                      onDragOver={(e) => e.preventDefault()}
                      onDrop={() =>
                        applyTaskOrder(
                          m.milestone_id,
                          m.milestone_id,
                          t.task_id,
                        )
                      }
                    >
                      {t.title} [{t.state}]
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )}
      </div>

      <AccessGate
        role={role}
        permission="planning.manage"
        fallback={<p>Read-only role. Planning actions disabled.</p>}
      >
        <div className="login-panel">
          <h3>Create Plan / Milestone / Task</h3>
          <div className="login-row">
            <input
              value={planName}
              onChange={(e) => setPlanName(e.target.value)}
              placeholder="plan name"
            />
            <button
              onClick={onCreatePlan}
              disabled={loading || !planName.trim()}
            >
              {loading ? "Saving..." : "Create Plan"}
            </button>
            <button
              onClick={onCreateMilestone}
              disabled={loading || !planID.trim()}
            >
              {loading ? "Saving..." : "Create Milestone"}
            </button>
            <button
              onClick={onCreateTask}
              disabled={loading || !milestoneID.trim()}
            >
              {loading ? "Saving..." : "Create Task"}
            </button>
          </div>
          <p>Milestone ID: {milestoneID || "-"}</p>
          <p>Task ID: {taskID || "-"}</p>
        </div>

        <div className="login-panel">
          <h3>Dependencies</h3>
          <div className="login-row">
            <input
              value={taskID}
              onChange={(e) => setTaskID(e.target.value)}
              placeholder="task id"
            />
            <input
              value={dependsOnTaskID}
              onChange={(e) => setDependsOnTaskID(e.target.value)}
              placeholder="depends on task id"
            />
            <button
              onClick={onAddDependency}
              disabled={loading || !taskID.trim() || !dependsOnTaskID.trim()}
            >
              {loading ? "Saving..." : "Add Dependency"}
            </button>
          </div>
        </div>

        <div className="login-panel">
          <h3>Reorder / Bulk Update</h3>
          <div className="login-row">
            <input
              value={orderedTaskIDs}
              onChange={(e) => setOrderedTaskIDs(e.target.value)}
              placeholder="ordered task ids (comma)"
            />
            <button
              onClick={onReorderTasks}
              disabled={
                loading || !orderedTaskIDs.trim() || !milestoneID.trim()
              }
            >
              {loading ? "Saving..." : "Reorder Tasks"}
            </button>
          </div>
          <div className="login-row">
            <input
              value={bulkTaskIDs}
              onChange={(e) => setBulkTaskIDs(e.target.value)}
              placeholder="task ids for bulk update"
            />
            <button
              onClick={onBulkUpdate}
              disabled={loading || !bulkTaskIDs.trim()}
            >
              {loading ? "Saving..." : "Bulk Mark Done"}
            </button>
          </div>
        </div>
      </AccessGate>
    </section>
  );
}

function messageFromError(e: unknown): string {
  if (typeof e === "object" && e && "message" in e) {
    return String((e as { message: string }).message);
  }
  return "Request failed";
}

function reorderByDrop<T extends { task_id: string }>(
  items: T[],
  draggedTaskID: string,
  targetTaskID: string,
): T[] {
  const draggedIndex = items.findIndex(
    (item) => item.task_id === draggedTaskID,
  );
  const targetIndex = items.findIndex((item) => item.task_id === targetTaskID);
  if (draggedIndex < 0 || targetIndex < 0 || draggedIndex === targetIndex) {
    return items;
  }
  const next = [...items];
  const [dragged] = next.splice(draggedIndex, 1);
  next.splice(targetIndex, 0, dragged);
  return next;
}
