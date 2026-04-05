import { useEffect, useMemo, useState } from "react";

import {
  getCohortFeatures,
  getDashboardOverview,
  getLearnerFeatures,
  getReportingMetrics,
  refreshDashboard,
  runNightlyFeatureBatch,
  getTodaySessions,
  type CohortFeature,
  type DashboardOverview,
  type LearnerFeature,
  type ReportingMetric,
  type TodaySession,
} from "../../api/endpoints";

import "./dashboard.css";

const REFRESH_MS = 20_000;

const kpiLabels: Record<string, string> = {
  enrollment_growth: "Enrollment",
  repeat_attendance: "Repeat Attendance",
  study_time_logged: "Study Time",
  content_conversion: "Content Conversion",
  community_activity: "Community Activity",
};

const kpiOrder = [
  "enrollment_growth",
  "repeat_attendance",
  "study_time_logged",
  "content_conversion",
  "community_activity",
];

type LoadState = {
  loading: boolean;
  error: string | null;
  overview: DashboardOverview | null;
  sessions: TodaySession[];
  lastUpdated: string | null;
  batchStatus: string | null;
  learnerFeatures: LearnerFeature[];
  cohortFeatures: CohortFeature[];
  reportingMetrics: ReportingMetric[];
};

export function DashboardPage() {
  const [state, setState] = useState<LoadState>({
    loading: true,
    error: null,
    overview: null,
    sessions: [],
    lastUpdated: null,
    batchStatus: null,
    learnerFeatures: [],
    cohortFeatures: [],
    reportingMetrics: [],
  });

  useEffect(() => {
    let active = true;

    const load = async () => {
      try {
        const [overview, sessions] = await Promise.all([
          getDashboardOverview(),
          getTodaySessions(),
        ]);
        if (!active) return;
        setState({
          loading: false,
          error: null,
          overview,
          sessions,
          lastUpdated: new Date().toLocaleTimeString(),
          batchStatus: null,
          learnerFeatures: [],
          cohortFeatures: [],
          reportingMetrics: [],
        });
      } catch (err) {
        if (!active) return;
        const message =
          typeof err === "object" && err && "message" in err
            ? String((err as { message: string }).message)
            : "Dashboard load failed";
        setState((prev) => ({ ...prev, loading: false, error: message }));
      }
    };

    load();
    const timer = window.setInterval(load, REFRESH_MS);

    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, []);

  const runFeatureBatch = async () => {
    try {
      const out = await runNightlyFeatureBatch();
      setState((prev) => ({
        ...prev,
        batchStatus: `Feature batch completed (${out.batch_ids.length} windows)`,
      }));
    } catch (err) {
      setState((prev) => ({
        ...prev,
        batchStatus: errorMessage(err),
      }));
    }
  };

  const loadFeatureViews = async () => {
    try {
      const [learners, cohorts, metrics] = await Promise.all([
        getLearnerFeatures({ windowDays: 30, limit: 5 }),
        getCohortFeatures({ windowDays: 30, limit: 5 }),
        getReportingMetrics({ windowDays: 30 }),
      ]);
      setState((prev) => ({
        ...prev,
        learnerFeatures: learners,
        cohortFeatures: cohorts,
        reportingMetrics: metrics,
      }));
    } catch (err) {
      setState((prev) => ({ ...prev, error: errorMessage(err) }));
    }
  };

  const refreshNow = async () => {
    try {
      await refreshDashboard();
      setState((prev) => ({
        ...prev,
        batchStatus: "Dashboard refresh queued",
      }));
    } catch (err) {
      setState((prev) => ({ ...prev, batchStatus: errorMessage(err) }));
    }
  };

  const orderedKpis = useMemo(() => {
    const list = state.overview?.kpis ?? [];
    const byKey = new Map(list.map((item) => [item.metric_key, item]));
    return kpiOrder
      .map((key) => byKey.get(key))
      .filter(Boolean) as DashboardOverview["kpis"];
  }, [state.overview]);

  return (
    <section className="dashboard-page">
      <header className="dashboard-header">
        <div>
          <p className="dashboard-kicker">TrainingOps Live Overview</p>
          <h1>Dashboard</h1>
        </div>
        <p className="dashboard-updated">Updated: {state.lastUpdated ?? "-"}</p>
      </header>

      {state.error ? (
        <div className="dashboard-error">{state.error}</div>
      ) : null}

      <div className="dashboard-grid">
        <section className="dashboard-panel dashboard-kpi-panel">
          <h2>KPI Tiles</h2>
          <div className="kpi-grid">
            {orderedKpis.map((kpi) => (
              <article key={kpi.metric_key} className="kpi-tile">
                <p className="kpi-label">
                  {kpiLabels[kpi.metric_key] ?? kpi.metric_key}
                </p>
                <p className="kpi-value">
                  {formatKpiValue(kpi.metric_key, kpi.metric_value)}
                </p>
              </article>
            ))}
          </div>
        </section>

        <section className="dashboard-panel">
          <h2>Pending Approvals</h2>
          <p className="single-metric">
            {state.overview?.summary.pending_approvals ?? 0}
          </p>
        </section>

        <section className="dashboard-panel dashboard-heatmap-panel">
          <h2>Occupancy Heatmap</h2>
          <div className="heatmap-grid">
            {(state.overview?.heatmap ?? []).map((cell, idx) => (
              <div
                key={`${cell.hour_bucket}-${cell.room_id ?? "all"}-${idx}`}
                className="heatmap-cell"
                style={{
                  opacity: Math.max(0.15, Math.min(1, cell.occupancy_rate)),
                }}
                title={`Hour ${pad2(cell.hour_bucket)} | Room ${cell.room_id ?? "all"} | Occupancy ${(cell.occupancy_rate * 100).toFixed(1)}%`}
              >
                <span>{pad2(cell.hour_bucket)}:00</span>
                <strong>{(cell.occupancy_rate * 100).toFixed(0)}%</strong>
              </div>
            ))}
          </div>
        </section>

        <section className="dashboard-panel dashboard-sessions-panel">
          <h2>Today's Sessions</h2>
          {state.loading ? <p>Loading...</p> : null}
          {!state.loading && state.sessions.length === 0 ? (
            <p>No sessions scheduled.</p>
          ) : null}
          {state.sessions.length > 0 ? (
            <ul className="sessions-list">
              {state.sessions.map((session) => (
                <li key={session.session_id} className="session-row">
                  <div>
                    <p className="session-title">{session.title}</p>
                    <p className="session-meta">
                      {formatTime(session.starts_at)} -{" "}
                      {formatTime(session.ends_at)} | Room {session.room_id}
                    </p>
                  </div>
                  <p className="session-occupancy">
                    {session.booked_seats}/{session.capacity} (
                    {(session.occupancy_rate * 100).toFixed(0)}%)
                  </p>
                </li>
              ))}
            </ul>
          ) : null}
        </section>

        <section className="dashboard-panel">
          <h2>Feature Store Ops</h2>
          <p>
            <button onClick={refreshNow}>Run Dashboard Refresh</button>{" "}
            <button onClick={runFeatureBatch}>Run Nightly Feature Batch</button>{" "}
            <button onClick={loadFeatureViews}>Load Feature Views</button>
          </p>
          {state.batchStatus ? <p>{state.batchStatus}</p> : null}
          <p>
            Learner features: {state.learnerFeatures.length} | Cohort features:{" "}
            {state.cohortFeatures.length}
          </p>
          {state.reportingMetrics.length > 0 ? (
            <ul>
              {state.reportingMetrics.slice(0, 3).map((metric) => (
                <li key={metric.metric_key}>
                  {metric.metric_key}: {(metric.metric_value * 100).toFixed(1)}%
                </li>
              ))}
            </ul>
          ) : (
            <p>No reporting metrics loaded.</p>
          )}
        </section>
      </div>
    </section>
  );
}

function errorMessage(err: unknown): string {
  return typeof err === "object" && err && "message" in err
    ? String((err as { message: string }).message)
    : "Request failed";
}

function formatKpiValue(metricKey: string, value: number): string {
  if (
    metricKey === "enrollment_growth" ||
    metricKey === "repeat_attendance" ||
    metricKey === "content_conversion"
  ) {
    return `${(value * 100).toFixed(1)}%`;
  }
  if (metricKey === "study_time_logged") {
    return `${Math.round(value)} min`;
  }
  return value.toFixed(1);
}

function formatTime(raw: string): string {
  return new Date(raw).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

function pad2(v: number): string {
  return String(v).padStart(2, "0");
}
