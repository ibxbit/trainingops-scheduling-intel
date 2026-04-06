import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { App } from "../App";
import { AdminPage } from "../features/admin/AdminPage";
import { useSessionStore } from "../state/session-store";

function json(status: number, payload: unknown) {
  return Promise.resolve(
    new Response(JSON.stringify(payload), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

describe("Admin route and flows", () => {
  beforeEach(() => {
    useSessionStore.setState({ user: null, isReady: false });
  });

  it("blocks non-admin access to /admin route", async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/v1/auth/me")) {
        return json(200, {
          data: {
            tenant_id: "t1",
            user_id: "u1",
            roles: ["learner"],
          },
        });
      }
      if (url.includes("/dashboard/overview")) {
        return json(200, {
          data: {
            summary: {
              metric_date: "2026-04-01",
              todays_sessions: 0,
              pending_approvals: 0,
            },
            kpis: [],
            heatmap: [],
          },
        });
      }
      if (url.includes("/dashboard/today-sessions")) {
        return json(200, { data: [] });
      }
      return json(404, { error: "not found" });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter initialEntries={["/admin"]}>
        <App />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(
        screen.getByText("You do not have access to this page."),
      ).toBeTruthy();
    });
  });

  it("shows validation, error, and success states for admin settings", async () => {
    useSessionStore.setState({
      user: {
        tenantId: "11111111-1111-1111-1111-111111111111",
        userId: "11111111-1111-1111-1111-111111111101",
        roles: ["administrator"],
        primaryRole: "administrator",
      },
      isReady: true,
    });

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (
        url.endsWith("/api/v1/admin/tenants") &&
        (!init?.method || init.method === "GET")
      ) {
        return json(200, {
          data: [
            {
              tenant_id: "11111111-1111-1111-1111-111111111111",
              tenant_slug: "acme-training",
              tenant_name: "Acme Training",
              allow_self_registration: false,
              require_mfa: false,
              max_active_bookings_per_learner: 3,
            },
          ],
        });
      }
      if (
        url.endsWith("/api/v1/admin/permissions/matrix") &&
        (!init?.method || init.method === "GET")
      ) {
        return json(200, {
          data: [
            {
              role: "administrator",
              permission: "tenant.settings.manage",
              allowed: true,
            },
          ],
        });
      }
      if (
        url.endsWith("/api/v1/admin/users/roles") &&
        (!init?.method || init.method === "GET")
      ) {
        return json(200, {
          data: [
            {
              user_id: "11111111-1111-1111-1111-111111111104",
              username: "learner1",
              roles: ["learner"],
            },
          ],
        });
      }
      if (
        url.includes("/api/v1/admin/tenants/") &&
        (init?.method === "PUT" || init?.method === undefined)
      ) {
        const body = init?.body ? JSON.parse(String(init.body)) : {};
        if (body.tenant_name === "Force Error") {
          return json(400, { error: "tenant settings save failed" });
        }
        return json(200, {
          data: {
            tenant_id: "11111111-1111-1111-1111-111111111111",
            tenant_slug: body.tenant_slug,
            tenant_name: body.tenant_name,
            allow_self_registration: false,
            require_mfa: false,
            max_active_bookings_per_learner: 3,
          },
        });
      }
      if (url.endsWith("/api/v1/admin/permissions/matrix") && init?.method === "PUT") {
        return json(200, { data: { status: "updated" } });
      }
      if (url.includes("/api/v1/admin/users/") && init?.method === "POST") {
        return json(200, { data: { status: "assigned" } });
      }
      if (url.includes("/api/v1/admin/users/") && init?.method === "DELETE") {
        return json(200, { data: { status: "revoked" } });
      }
      return json(404, { error: "not found" });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<AdminPage />);

    await waitFor(() => {
      expect(screen.getByText("Administrator data loaded")).toBeTruthy();
    });

    const slugInput = screen.getByPlaceholderText("tenant slug");
    const nameInput = screen.getByPlaceholderText("tenant name");
    await userEvent.clear(slugInput);
    await userEvent.click(screen.getByRole("button", { name: "Save Settings" }));
    expect(screen.getByText("Tenant name and slug are required")).toBeTruthy();

    await userEvent.type(slugInput, "acme-training");
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "Force Error");
    await userEvent.click(screen.getByRole("button", { name: "Save Settings" }));
    await waitFor(() => {
      expect(screen.getByText("tenant settings save failed")).toBeTruthy();
    });

    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "Acme Training Success");
    await userEvent.click(screen.getByRole("button", { name: "Save Settings" }));
    await waitFor(() => {
      expect(screen.getByText("Tenant settings saved")).toBeTruthy();
    });
  });
});
