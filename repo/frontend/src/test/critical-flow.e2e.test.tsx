import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { App } from "../App";
import { useSessionStore } from "../state/session-store";

describe("Critical user journey (frontend e2e style)", () => {
  beforeEach(() => {
    useSessionStore.setState({ user: null, isReady: false });
  });

  it("logs in and places a booking hold", async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/api/v1/auth/me") && !init?.method) {
        return Promise.resolve(
          new Response(JSON.stringify({ error: "not authenticated" }), {
            status: 401,
          }),
        );
      }
      if (url.endsWith("/api/v1/auth/login")) {
        return Promise.resolve(
          new Response(JSON.stringify({ data: { status: "authenticated" } }), {
            status: 200,
          }),
        );
      }
      if (url.endsWith("/api/v1/auth/me")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              data: { tenant_id: "t1", user_id: "u1", roles: ["learner"] },
            }),
            { status: 200 },
          ),
        );
      }
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
      if (url.includes("/bookings/hold")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              data: {
                booking_id: "b1",
                state: "held",
                hold_expires_at: new Date(Date.now() + 240000).toISOString(),
                reschedule_count: 0,
              },
            }),
            { status: 201 },
          ),
        );
      }
      if (url.includes("/calendar/availability/")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({ data: { reason: "available", alternatives: [] } }),
            { status: 200 },
          ),
        );
      }
      return Promise.resolve(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter initialEntries={["/login"]}>
        <App />
      </MemoryRouter>,
    );

    await userEvent.click(screen.getByRole("button", { name: "Login" }));

    await waitFor(() => {
      expect(screen.getByText("Learner session")).toBeTruthy();
    });

    await userEvent.click(screen.getByRole("button", { name: "Booking" }));
    await userEvent.type(
      screen.getByPlaceholderText("session uuid"),
      "session-1",
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Place 5-Min Hold" }),
    );

    await waitFor(() => {
      expect(screen.getByText("Hold placed successfully.")).toBeTruthy();
    });
  });
});
