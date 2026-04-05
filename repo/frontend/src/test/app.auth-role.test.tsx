import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { App } from "../App";
import { useSessionStore } from "../state/session-store";

function ok(data: unknown) {
  return Promise.resolve(
    new Response(JSON.stringify({ data }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

function err(status: number, error: string) {
  return Promise.resolve(
    new Response(JSON.stringify({ error }), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

describe("App auth and role derivation", () => {
  beforeEach(() => {
    useSessionStore.setState({ user: null, isReady: false });
  });

  it("derives role from backend identity and never from UI input", async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/api/v1/auth/me") && !init?.method) {
        return err(401, "not authenticated");
      }
      if (url.endsWith("/api/v1/auth/login")) {
        return ok({ status: "authenticated" });
      }
      if (url.endsWith("/api/v1/auth/me")) {
        return ok({
          tenant_id: "tenant-1",
          user_id: "user-1",
          roles: ["learner", "administrator"],
        });
      }
      if (url.includes("/dashboard/overview")) {
        return ok({
          summary: {
            metric_date: "2026-04-01",
            todays_sessions: 0,
            pending_approvals: 0,
          },
          kpis: [],
          heatmap: [],
        });
      }
      if (url.includes("/dashboard/today-sessions")) {
        return ok([]);
      }
      return err(404, "not found");
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter initialEntries={["/login"]}>
        <App />
      </MemoryRouter>,
    );

    await userEvent.click(screen.getByRole("button", { name: "Login" }));

    await waitFor(() => {
      expect(screen.getByText("Administrator session")).toBeTruthy();
    });
    expect(screen.queryByText("UI Role")).toBeNull();
  });
});
