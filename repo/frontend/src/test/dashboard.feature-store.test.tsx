import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { DashboardPage } from "../features/dashboard/DashboardPage";

describe("Dashboard feature store actions", () => {
  it("runs batch and loads feature views", async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/dashboard/overview")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              data: {
                summary: {
                  metric_date: "2026-04-01",
                  todays_sessions: 1,
                  pending_approvals: 0,
                },
                kpis: [],
                heatmap: [],
              },
            }),
            { status: 200 },
          ),
        );
      }
      if (url.includes("/dashboard/today-sessions")) {
        return Promise.resolve(
          new Response(JSON.stringify({ data: [] }), { status: 200 }),
        );
      }
      if (
        url.endsWith("/dashboard/feature-store/nightly-batch") &&
        init?.method === "POST"
      ) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ data: { batch_ids: ["b1", "b2", "b3"] } }),
            { status: 200 },
          ),
        );
      }
      if (url.includes("/dashboard/feature-store/learners")) {
        return Promise.resolve(
          new Response(JSON.stringify({ data: [{ learner_user_id: "u1" }] }), {
            status: 200,
          }),
        );
      }
      if (url.includes("/dashboard/feature-store/cohorts")) {
        return Promise.resolve(
          new Response(JSON.stringify({ data: [{ cohort_id: "c1" }] }), {
            status: 200,
          }),
        );
      }
      if (url.includes("/dashboard/feature-store/reporting-metrics")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              data: [
                {
                  metric_key: "attendance_rate",
                  metric_value: 0.8,
                  numerator: 8,
                  denominator: 10,
                },
              ],
            }),
            { status: 200 },
          ),
        );
      }
      return Promise.resolve(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<DashboardPage />);

    await userEvent.click(
      screen.getByRole("button", { name: "Run Nightly Feature Batch" }),
    );
    await waitFor(() => {
      expect(
        screen.getByText("Feature batch completed (3 windows)"),
      ).toBeTruthy();
    });

    await userEvent.click(
      screen.getByRole("button", { name: "Load Feature Views" }),
    );
    await waitFor(() => {
      expect(
        screen.getByText("Learner features: 1 | Cohort features: 1"),
      ).toBeTruthy();
      expect(screen.getByText("attendance_rate: 80.0%")).toBeTruthy();
    });
  });
});
