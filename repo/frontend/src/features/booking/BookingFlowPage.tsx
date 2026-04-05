import { useEffect, useMemo, useState } from "react";

import {
  cancelBooking,
  confirmBooking,
  createBookingHold,
  getAvailability,
  rescheduleBooking,
  type AlternativeSession,
  type AvailabilityReason,
  type HoldResponse,
} from "../../api/endpoints";
import { canAccess } from "../../auth/policy";
import { useSessionStore } from "../../state/session-store";

import "./booking.css";

type ConflictInfo = {
  error: string;
  reason?: AvailabilityReason;
  alternatives?: AlternativeSession[];
};

const reasonLabels: Record<AvailabilityReason, string> = {
  available: "Available",
  outside_academic_calendar: "Outside academic calendar",
  blackout_date: "Blackout date",
  outside_allowed_slot: "Outside allowed slot",
  room_occupied: "Room occupied",
  instructor_unavailable: "Instructor unavailable",
  capacity_reached: "Capacity reached",
};

export function BookingFlowPage() {
  const role = useSessionStore((s) => s.user?.primaryRole ?? null);
  const canManageBooking = canAccess(role, "booking.manage");

  const [sessionId, setSessionId] = useState("");
  const [bookingId, setBookingId] = useState("");
  const [reason, setReason] = useState("");
  const [rescheduleSessionId, setRescheduleSessionId] = useState("");

  const [availabilityReason, setAvailabilityReason] =
    useState<AvailabilityReason | null>(null);
  const [alternatives, setAlternatives] = useState<AlternativeSession[]>([]);
  const [hold, setHold] = useState<HoldResponse | null>(null);
  const [conflict, setConflict] = useState<ConflictInfo | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const holdRemaining = useMemo(() => {
    if (!hold?.hold_expires_at) return null;
    const ms = new Date(hold.hold_expires_at).getTime() - now;
    if (ms <= 0) return "Expired";
    const total = Math.floor(ms / 1000);
    const m = Math.floor(total / 60);
    const s = total % 60;
    return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  }, [hold, now]);

  const loadAvailability = async () => {
    if (!sessionId.trim() || busy) return;
    setBusy(true);
    setStatusMessage(null);
    setConflict(null);
    try {
      const data = await getAvailability(sessionId.trim());
      setAvailabilityReason(data.reason);
      setAlternatives(data.alternatives.slice(0, 3));
    } catch (err) {
      setStatusMessage(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const onHold = async () => {
    if (!sessionId.trim() || busy) return;
    setBusy(true);
    setStatusMessage(null);
    setConflict(null);
    try {
      const data = await createBookingHold({
        session_id: sessionId.trim(),
        ...(reason ? { reason } : {}),
      });
      setHold(data);
      setBookingId(data.booking_id);
      setStatusMessage("Hold placed successfully.");
    } catch (err) {
      const parsed = parseConflict(err);
      setConflict(parsed);
      setStatusMessage(parsed.error);
      if (parsed.reason) setAvailabilityReason(parsed.reason);
      if (parsed.alternatives) setAlternatives(parsed.alternatives.slice(0, 3));
    } finally {
      setBusy(false);
    }
  };

  const onConfirm = async () => {
    if (!bookingId.trim() || busy) return;
    setBusy(true);
    setStatusMessage(null);
    setConflict(null);
    try {
      await confirmBooking(bookingId.trim(), reason || undefined);
      setStatusMessage("Booking confirmed.");
      setHold((prev) =>
        prev ? { ...prev, state: "confirmed", hold_expires_at: null } : prev,
      );
    } catch (err) {
      setStatusMessage(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const onReschedule = async () => {
    if (!bookingId.trim() || !rescheduleSessionId.trim() || busy) return;
    setBusy(true);
    setStatusMessage(null);
    setConflict(null);
    try {
      await rescheduleBooking(
        bookingId.trim(),
        rescheduleSessionId.trim(),
        reason || undefined,
      );
      setStatusMessage("Booking rescheduled.");
      setSessionId(rescheduleSessionId.trim());
      setHold((prev) =>
        prev
          ? {
              ...prev,
              state: "rescheduled",
            }
          : prev,
      );
    } catch (err) {
      const parsed = parseConflict(err);
      setConflict(parsed);
      setStatusMessage(parsed.error);
      if (parsed.reason) setAvailabilityReason(parsed.reason);
      if (parsed.alternatives) setAlternatives(parsed.alternatives.slice(0, 3));
    } finally {
      setBusy(false);
    }
  };

  const onCancel = async () => {
    if (!bookingId.trim() || busy) return;
    setBusy(true);
    setStatusMessage(null);
    setConflict(null);
    try {
      await cancelBooking(bookingId.trim(), reason || undefined);
      setStatusMessage("Booking canceled.");
      setHold((prev) =>
        prev ? { ...prev, state: "canceled", hold_expires_at: null } : prev,
      );
    } catch (err) {
      setStatusMessage(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="booking-page">
      <header className="booking-header">
        <h1>Booking Flow</h1>
        <p>Uses backend validation and conflict analysis directly.</p>
      </header>

      <div className="booking-layout">
        <div className="booking-card">
          <h2>Session Availability</h2>
          <label>
            Session ID
            <input
              value={sessionId}
              onChange={(e) => setSessionId(e.target.value)}
              placeholder="session uuid"
            />
          </label>
          <button
            onClick={loadAvailability}
            disabled={busy || !sessionId.trim()}
          >
            Check Availability
          </button>
          <p className="muted">
            Reason:{" "}
            {availabilityReason ? reasonLabels[availabilityReason] : "-"}
          </p>
          {alternatives.length > 0 ? (
            <AlternativeList
              alternatives={alternatives}
              onPick={setRescheduleSessionId}
            />
          ) : null}
        </div>

        <div className="booking-card">
          <h2>Hold / Manage Booking</h2>
          {!canManageBooking ? (
            <p className="muted">
              Your role is read-only for hold, confirm, reschedule, and cancel
              actions.
            </p>
          ) : null}
          <label>
            Booking ID
            <input
              value={bookingId}
              onChange={(e) => setBookingId(e.target.value)}
              placeholder="booking uuid"
            />
          </label>
          <label>
            Reason (optional)
            <input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="optional note"
            />
          </label>

          <div className="actions-row">
            <button
              onClick={onHold}
              disabled={!canManageBooking || busy || !sessionId.trim()}
            >
              Place 5-Min Hold
            </button>
            <button
              onClick={onConfirm}
              disabled={!canManageBooking || busy || !bookingId.trim()}
            >
              Confirm
            </button>
            <button
              onClick={onCancel}
              disabled={!canManageBooking || busy || !bookingId.trim()}
            >
              Cancel
            </button>
          </div>

          <label>
            Reschedule to Session ID
            <input
              value={rescheduleSessionId}
              onChange={(e) => setRescheduleSessionId(e.target.value)}
              placeholder="new session uuid"
            />
          </label>
          <button
            onClick={onReschedule}
            disabled={
              !canManageBooking ||
              busy ||
              !bookingId.trim() ||
              !rescheduleSessionId.trim()
            }
          >
            Reschedule
          </button>

          <div className="policy-panel">
            <p>
              Hold timer: <strong>{holdRemaining ?? "-"}</strong>
            </p>
            <p>
              Booking state: <strong>{hold?.state ?? "-"}</strong>
            </p>
            <p>
              Reschedule usage:{" "}
              <strong>{hold ? `${hold.reschedule_count}/2` : "-"}</strong> (max
              2)
            </p>
            <p>
              Cancellation policy:{" "}
              <strong>Not allowed within 24 hours before session start</strong>
            </p>
          </div>
        </div>

        <div className="booking-card">
          <h2>Conflict Details</h2>
          <p>{statusMessage ?? "No recent action."}</p>
          {conflict ? (
            <>
              <p>
                Error: <strong>{conflict.error}</strong>
              </p>
              <p>
                Reason: {conflict.reason ? reasonLabels[conflict.reason] : "-"}
              </p>
              {conflict.alternatives?.length ? (
                <AlternativeList
                  alternatives={conflict.alternatives}
                  onPick={setRescheduleSessionId}
                />
              ) : null}
            </>
          ) : null}
        </div>
      </div>
    </section>
  );
}

function AlternativeList({
  alternatives,
  onPick,
}: {
  alternatives: AlternativeSession[];
  onPick: (sessionId: string) => void;
}) {
  return (
    <div className="alternatives">
      <p>Alternative suggestions (max 3):</p>
      <ul>
        {alternatives.slice(0, 3).map((alt) => (
          <li key={alt.session_id}>
            <button onClick={() => onPick(alt.session_id)}>
              {alt.session_id}
            </button>
            <span>
              {new Date(alt.starts_at).toLocaleString()} -{" "}
              {new Date(alt.ends_at).toLocaleTimeString()} | Room {alt.room_id}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function errorMessage(err: unknown): string {
  if (typeof err === "object" && err && "message" in err) {
    return String((err as { message: string }).message);
  }
  return "Request failed";
}

function parseConflict(err: unknown): ConflictInfo {
  if (typeof err !== "object" || !err) {
    return { error: "Request failed" };
  }
  const raw = err as {
    message?: string;
    reason?: AvailabilityReason;
    alternatives?: AlternativeSession[];
  };
  return {
    error: raw.message ?? "Booking conflict",
    reason: raw.reason,
    alternatives: raw.alternatives,
  };
}
