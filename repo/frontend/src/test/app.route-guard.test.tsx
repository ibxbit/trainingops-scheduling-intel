import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { App } from "../App";
import { useSessionStore } from "../state/session-store";

describe("Route guards", () => {
  beforeEach(() => {
    useSessionStore.setState({ user: null, isReady: false });
  });

  it("redirects unauthenticated users to login for protected routes", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith("/api/v1/auth/me")) {
          return Promise.resolve(
            new Response(JSON.stringify({ error: "not authenticated" }), {
              status: 401,
            }),
          );
        }
        return Promise.resolve(
          new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
        );
      }),
    );

    render(
      <MemoryRouter initialEntries={["/dashboard"]}>
        <App />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Login" })).toBeTruthy();
    });
  });
});
