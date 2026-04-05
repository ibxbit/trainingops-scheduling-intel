import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { PlanningPage } from "../features/planning/PlanningPage";
import { useSessionStore } from "../state/session-store";

describe("Planning drag-and-drop reorder", () => {
  beforeEach(() => {
    useSessionStore.setState({
      user: {
        userId: "u1",
        tenantId: "t1",
        roles: ["program_coordinator"],
        primaryRole: "program_coordinator",
      },
      isReady: true,
    });
  });

  it("reorders tasks within the same milestone", async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/planning/plans/") && url.endsWith("/tree")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              data: {
                plan: { plan_id: "p1", name: "Plan" },
                milestones: [
                  { milestone_id: "m1", title: "M1", sort_order: 0 },
                ],
                tasks: [
                  {
                    task_id: "t1",
                    milestone_id: "m1",
                    title: "Task 1",
                    state: "todo",
                    estimated_minutes: 15,
                    actual_minutes: 0,
                    sort_order: 0,
                  },
                  {
                    task_id: "t2",
                    milestone_id: "m1",
                    title: "Task 2",
                    state: "todo",
                    estimated_minutes: 15,
                    actual_minutes: 0,
                    sort_order: 1,
                  },
                ],
              },
            }),
            { status: 200 },
          ),
        );
      }
      if (
        url.includes("/planning/milestones/m1/reorder-tasks") &&
        init?.method === "PATCH"
      ) {
        return Promise.resolve(
          new Response(JSON.stringify({ data: { status: "ok" } }), {
            status: 200,
          }),
        );
      }
      return Promise.resolve(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<PlanningPage />);

    await userEvent.type(screen.getByPlaceholderText("plan id"), "p1");
    await userEvent.click(screen.getByRole("button", { name: "Load Tree" }));

    await waitFor(() => {
      expect(screen.getByText("Task 1 [todo]")).toBeTruthy();
      expect(screen.getByText("Task 2 [todo]")).toBeTruthy();
    });

    const task1 = screen.getByText("Task 1 [todo]");
    const task2 = screen.getByText("Task 2 [todo]");
    fireEvent.dragStart(task1);
    fireEvent.dragOver(task2);
    fireEvent.drop(task2);

    await waitFor(() => {
      expect(screen.getByText("Task order updated")).toBeTruthy();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/planning/milestones/m1/reorder-tasks",
      expect.objectContaining({ method: "PATCH" }),
    );
  });
});
