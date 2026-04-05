import { useState } from "react";

import {
  createAcademicTerm,
  createBlackoutDate,
  createTimeSlotRule,
  getAvailability,
  type AvailabilityReason,
} from "../../api/endpoints";
import { AccessGate } from "../../auth/access-control";
import { useSessionStore } from "../../state/session-store";

const reasonLabels: Record<AvailabilityReason, string> = {
  available: "Available",
  outside_academic_calendar: "Outside academic calendar",
  blackout_date: "Blackout date",
  outside_allowed_slot: "Outside allowed slot",
  room_occupied: "Room occupied",
  instructor_unavailable: "Instructor unavailable",
  capacity_reached: "Capacity reached",
};

export function CalendarPage() {
  const role = useSessionStore((s) => s.user?.primaryRole ?? null);
  const [sessionID, setSessionID] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [availability, setAvailability] = useState<AvailabilityReason | null>(
    null,
  );
  const [alternatives, setAlternatives] = useState<
    Array<{
      session_id: string;
      starts_at: string;
      ends_at: string;
      room_id: string;
    }>
  >([]);
  const [status, setStatus] = useState<string | null>(null);

  const [slotForm, setSlotForm] = useState({
    weekday: 1,
    slot_start: "10:00",
    slot_end: "11:00",
  });
  const [blackoutForm, setBlackoutForm] = useState({
    blackout_date: "",
    reason: "maintenance",
  });
  const [termForm, setTermForm] = useState({
    name: "Term",
    start_date: "",
    end_date: "",
  });

  const checkAvailability = async () => {
    if (!sessionID.trim()) {
      setError("Session ID is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const data = await getAvailability(sessionID.trim());
      setAvailability(data.reason);
      setAlternatives(data.alternatives);
    } catch (e) {
      setError(messageFromError(e));
      setAvailability(null);
      setAlternatives([]);
    } finally {
      setLoading(false);
    }
  };

  const createSlotRule = async () => {
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await createTimeSlotRule({ ...slotForm, is_active: true });
      setStatus("Time slot rule created");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const createBlackout = async () => {
    if (!blackoutForm.blackout_date) {
      setError("Blackout date is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await createBlackoutDate({ ...blackoutForm, is_active: true });
      setStatus("Blackout date created");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const createTerm = async () => {
    if (!termForm.start_date || !termForm.end_date) {
      setError("Start and end dates are required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await createAcademicTerm({ ...termForm, is_active: true });
      setStatus("Academic term created");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <section>
      <h2>Calendar</h2>
      <p>Check availability and manage calendar controls.</p>
      {error ? <p className="error">{error}</p> : null}
      {status ? <p>{status}</p> : null}

      <div className="login-panel">
        <h3>Availability</h3>
        <div className="login-row">
          <input
            value={sessionID}
            onChange={(e) => setSessionID(e.target.value)}
            placeholder="session id"
          />
          <button
            onClick={checkAvailability}
            disabled={loading || !sessionID.trim()}
          >
            {loading ? "Checking..." : "Check"}
          </button>
        </div>
        {availability ? (
          <p>Result: {reasonLabels[availability]}</p>
        ) : (
          <p>No availability check yet.</p>
        )}
        {alternatives.length > 0 ? (
          <ul>
            {alternatives.map((alt) => (
              <li key={alt.session_id}>
                {alt.session_id} | {new Date(alt.starts_at).toLocaleString()} |
                room {alt.room_id}
              </li>
            ))}
          </ul>
        ) : (
          <p>No alternatives.</p>
        )}
      </div>

      <AccessGate
        role={role}
        permission="calendar.manage"
        fallback={<p>Read-only role. Management actions disabled.</p>}
      >
        <div className="login-panel">
          <h3>Create Time Slot Rule</h3>
          <div className="login-row">
            <input
              type="number"
              min={0}
              max={6}
              value={slotForm.weekday}
              onChange={(e) =>
                setSlotForm((prev) => ({
                  ...prev,
                  weekday: Number(e.target.value),
                }))
              }
            />
            <input
              value={slotForm.slot_start}
              onChange={(e) =>
                setSlotForm((prev) => ({ ...prev, slot_start: e.target.value }))
              }
              placeholder="HH:MM"
            />
            <input
              value={slotForm.slot_end}
              onChange={(e) =>
                setSlotForm((prev) => ({ ...prev, slot_end: e.target.value }))
              }
              placeholder="HH:MM"
            />
            <button onClick={createSlotRule} disabled={loading}>
              {loading ? "Saving..." : "Create Rule"}
            </button>
          </div>
        </div>

        <div className="login-panel">
          <h3>Create Blackout Date</h3>
          <div className="login-row">
            <input
              type="date"
              value={blackoutForm.blackout_date}
              onChange={(e) =>
                setBlackoutForm((prev) => ({
                  ...prev,
                  blackout_date: e.target.value,
                }))
              }
            />
            <input
              value={blackoutForm.reason}
              onChange={(e) =>
                setBlackoutForm((prev) => ({ ...prev, reason: e.target.value }))
              }
              placeholder="reason"
            />
            <button onClick={createBlackout} disabled={loading}>
              {loading ? "Saving..." : "Create Blackout"}
            </button>
          </div>
        </div>

        <div className="login-panel">
          <h3>Create Academic Term</h3>
          <div className="login-row">
            <input
              value={termForm.name}
              onChange={(e) =>
                setTermForm((prev) => ({ ...prev, name: e.target.value }))
              }
              placeholder="term name"
            />
            <input
              type="date"
              value={termForm.start_date}
              onChange={(e) =>
                setTermForm((prev) => ({ ...prev, start_date: e.target.value }))
              }
            />
            <input
              type="date"
              value={termForm.end_date}
              onChange={(e) =>
                setTermForm((prev) => ({ ...prev, end_date: e.target.value }))
              }
            />
            <button onClick={createTerm} disabled={loading}>
              {loading ? "Saving..." : "Create Term"}
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
