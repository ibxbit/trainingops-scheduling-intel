import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { App } from "../App";
import { useSessionStore } from "../state/session-store";
import {
  buildUploadResumeKey,
  clearUploadResumeCache,
} from "../state/upload-resume-cache";

function json(status: number, payload: unknown) {
  return Promise.resolve(
    new Response(JSON.stringify(payload), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

describe("Upload resume state isolation", () => {
  beforeEach(() => {
    localStorage.clear();
    useSessionStore.setState({ user: null, isReady: false });
  });

  it("namespaces resume keys by tenant and user", () => {
    const keyA = buildUploadResumeKey("tenant-1", "user-1", "notes.pdf", 100, "");
    const keyB = buildUploadResumeKey("tenant-1", "user-2", "notes.pdf", 100, "");
    expect(keyA).not.toEqual(keyB);
  });

  it("clears only active user resume cache on logout", async () => {
    const keyUser1 = buildUploadResumeKey(
      "tenant-1",
      "user-1",
      "notes.pdf",
      100,
      "doc-1",
    );
    const keyUser2 = buildUploadResumeKey(
      "tenant-1",
      "user-2",
      "notes.pdf",
      100,
      "doc-1",
    );
    localStorage.setItem(keyUser1, JSON.stringify({ uploadID: "u1", nextIndex: 1 }));
    localStorage.setItem(keyUser2, JSON.stringify({ uploadID: "u2", nextIndex: 2 }));

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/api/v1/auth/me")) {
        return json(200, {
          data: {
            tenant_id: "tenant-1",
            user_id: "user-1",
            roles: ["administrator"],
          },
        });
      }
      if (url.endsWith("/api/v1/auth/logout") && init?.method === "POST") {
        return json(200, { data: { status: "logged_out" } });
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
      <MemoryRouter initialEntries={["/dashboard"]}>
        <App />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByText("Administrator session")).toBeTruthy();
    });

    await userEvent.click(screen.getByRole("button", { name: "Logout" }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Login" })).toBeTruthy();
    });

    expect(localStorage.getItem(keyUser1)).toBeNull();
    expect(localStorage.getItem(keyUser2)).not.toBeNull();

    clearUploadResumeCache();
  });
});
