import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { BookingFlowPage } from "../features/booking/BookingFlowPage";
import { useSessionStore } from "../state/session-store";

describe("Booking page failure and race-safe behavior", () => {
  beforeEach(() => {
    useSessionStore.setState({
      user: {
        userId: "u1",
        tenantId: "t1",
        roles: ["learner"],
        primaryRole: "learner",
      },
      isReady: true,
    });
  });

  it("shows backend validation errors and ignores duplicate submit", async () => {
    const holdCall = vi.fn(async () => {
      await new Promise((r) => setTimeout(r, 50));
      return new Response(
        JSON.stringify({
          error: "capacity reached",
          reason: "capacity_reached",
          alternatives: [],
        }),
        { status: 409 },
      );
    });

    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/bookings/hold") && init?.method === "POST") {
        return holdCall();
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

    render(<BookingFlowPage />);

    await userEvent.type(
      screen.getByPlaceholderText("session uuid"),
      "session-1",
    );
    const holdButton = screen.getByRole("button", { name: "Place 5-Min Hold" });
    await Promise.all([
      userEvent.click(holdButton),
      userEvent.click(holdButton),
    ]);

    await waitFor(() => {
      expect(screen.getAllByText("capacity reached").length).toBeGreaterThan(0);
    });
    expect(holdCall).toHaveBeenCalledTimes(1);
  });
});
