import { apiRequest } from "./http-client";

export type DashboardOverview = {
  summary: {
    metric_date: string;
    todays_sessions: number;
    pending_approvals: number;
  };
  kpis: Array<{
    metric_key: string;
    metric_value: number;
    numerator: number;
    denominator: number;
  }>;
  heatmap: Array<{
    hour_bucket: number;
    room_id: string | null;
    sessions_count: number;
    booked_seats: number;
    total_seats: number;
    occupancy_rate: number;
  }>;
};

export type TodaySession = {
  session_id: string;
  title: string;
  starts_at: string;
  ends_at: string;
  room_id: string;
  capacity: number;
  booked_seats: number;
  occupancy_rate: number;
  instructor_user_id: string | null;
};

export type AvailabilityReason =
  | "available"
  | "outside_academic_calendar"
  | "blackout_date"
  | "outside_allowed_slot"
  | "room_occupied"
  | "instructor_unavailable"
  | "capacity_reached";

export type AlternativeSession = {
  session_id: string;
  room_id: string;
  starts_at: string;
  ends_at: string;
};

export type AvailabilityResponse = {
  reason: AvailabilityReason;
  alternatives: AlternativeSession[];
};

export type HoldResponse = {
  booking_id: string;
  state: "held" | "confirmed" | "rescheduled" | "canceled" | "checked_in";
  hold_expires_at: string | null;
  reschedule_count: number;
};

export type CalendarRulePayload = {
  room_id?: string;
  weekday: number;
  slot_start: string;
  slot_end: string;
  is_active: boolean;
};

export type BlackoutPayload = {
  room_id?: string;
  blackout_date: string;
  reason: string;
  is_active: boolean;
};

export type TermPayload = {
  name: string;
  start_date: string;
  end_date: string;
  is_active: boolean;
};

export type DocumentSummary = {
  document_id: string;
  title: string;
  summary?: string;
  difficulty?: number;
  duration_minutes?: number;
};

export type PlanTree = {
  plan: {
    plan_id: string;
    name: string;
  };
  milestones: Array<{
    milestone_id: string;
    title: string;
    due_date?: string;
    sort_order: number;
  }>;
  tasks: Array<{
    task_id: string;
    milestone_id: string;
    title: string;
    state: string;
    due_at?: string;
    estimated_minutes: number;
    actual_minutes: number;
    sort_order: number;
  }>;
};

export type LearnerFeature = {
  learner_user_id: string;
  feature_date: string;
  window_days: number;
  cohort_id: string | null;
  study_time_minutes: number;
  attendance_count: number;
  attendance_rate: number;
  completion_rate: number;
  community_events_count: number;
  content_download_count: number;
  content_share_count: number;
  at_risk_score: number;
  segment_label: string | null;
};

export type CohortFeature = {
  cohort_id: string;
  feature_date: string;
  window_days: number;
  learner_count: number;
  attendance_rate_avg: number;
  completion_rate_avg: number;
  at_risk_ratio: number;
  content_conversion_avg: number;
};

export type ReportingMetric = {
  metric_key: string;
  numerator: number;
  denominator: number;
  metric_value: number;
};

export type UploadSession = {
  upload_id: string;
  document_id?: string;
  file_name: string;
  mime_type: string;
  total_chunks: number;
  chunk_size_bytes: number;
  expires_at: string;
  completed_at?: string | null;
};

export type DocumentVersion = {
  document_version_id: string;
  document_id: string;
  version_no: number;
  file_name: string;
  storage_path: string;
  mime_type: string;
  file_size_bytes: number;
  sha256_checksum: string;
  created_at: string;
};

export function getDashboardOverview(date?: string) {
  const q = date ? `?date=${encodeURIComponent(date)}` : "";
  return apiRequest<DashboardOverview>(`/dashboard/overview${q}`);
}

export function refreshDashboard(date?: string) {
  return apiRequest<{ refresh_id: string }>(`/dashboard/refresh`, {
    method: "POST",
    body: date ? { date } : {},
  });
}

export function runNightlyFeatureBatch(date?: string) {
  return apiRequest<{ batch_ids: string[] }>(
    `/dashboard/feature-store/nightly-batch`,
    {
      method: "POST",
      body: date ? { date } : {},
    },
  );
}

export function getLearnerFeatures(params: {
  windowDays: 7 | 30 | 90;
  limit?: number;
  date?: string;
  segment?: string;
}) {
  const query = new URLSearchParams();
  query.set("window_days", String(params.windowDays));
  if (params.limit !== undefined) query.set("limit", String(params.limit));
  if (params.date) query.set("date", params.date);
  if (params.segment) query.set("segment", params.segment);
  return apiRequest<LearnerFeature[]>(
    `/dashboard/feature-store/learners?${query.toString()}`,
  );
}

export function getCohortFeatures(params: {
  windowDays: 7 | 30 | 90;
  limit?: number;
  date?: string;
}) {
  const query = new URLSearchParams();
  query.set("window_days", String(params.windowDays));
  if (params.limit !== undefined) query.set("limit", String(params.limit));
  if (params.date) query.set("date", params.date);
  return apiRequest<CohortFeature[]>(
    `/dashboard/feature-store/cohorts?${query.toString()}`,
  );
}

export function getReportingMetrics(params: {
  windowDays: 7 | 30 | 90;
  date?: string;
}) {
  const query = new URLSearchParams();
  query.set("window_days", String(params.windowDays));
  if (params.date) query.set("date", params.date);
  return apiRequest<ReportingMetric[]>(
    `/dashboard/feature-store/reporting-metrics?${query.toString()}`,
  );
}

export function getTodaySessions(date?: string, limit = 50) {
  const params = new URLSearchParams();
  if (date) params.set("date", date);
  params.set("limit", String(limit));
  return apiRequest<TodaySession[]>(
    `/dashboard/today-sessions?${params.toString()}`,
  );
}

export function createBookingHold(payload: Record<string, unknown>) {
  return apiRequest<HoldResponse>(`/bookings/hold`, {
    method: "POST",
    body: payload,
  });
}

export function getAvailability(sessionId: string) {
  return apiRequest<AvailabilityResponse>(
    `/calendar/availability/${encodeURIComponent(sessionId)}`,
  );
}

export function confirmBooking(bookingId: string, reason?: string) {
  return apiRequest<{ status: string }>(
    `/bookings/${encodeURIComponent(bookingId)}/confirm`,
    {
      method: "POST",
      body: reason ? { reason } : {},
    },
  );
}

export function rescheduleBooking(
  bookingId: string,
  sessionId: string,
  reason?: string,
) {
  return apiRequest<{ status: string }>(
    `/bookings/${encodeURIComponent(bookingId)}/reschedule`,
    {
      method: "POST",
      body: { session_id: sessionId, ...(reason ? { reason } : {}) },
    },
  );
}

export function cancelBooking(bookingId: string, reason?: string) {
  return apiRequest<{ status: string }>(
    `/bookings/${encodeURIComponent(bookingId)}/cancel`,
    {
      method: "POST",
      body: reason ? { reason } : {},
    },
  );
}

export function searchContent(query: string, limit = 25) {
  const q = `?q=${encodeURIComponent(query)}&limit=${limit}`;
  return apiRequest<DocumentSummary[]>(`/content/documents/search${q}`);
}

export function getPlanTree(planId: string) {
  return apiRequest<PlanTree>(
    `/planning/plans/${encodeURIComponent(planId)}/tree`,
  );
}

export function createTimeSlotRule(payload: CalendarRulePayload) {
  return apiRequest<{ rule_id: string }>(`/calendar/time-slots`, {
    method: "POST",
    body: payload,
  });
}

export function createBlackoutDate(payload: BlackoutPayload) {
  return apiRequest<{ blackout_id: string }>(`/calendar/blackouts`, {
    method: "POST",
    body: payload,
  });
}

export function createAcademicTerm(payload: TermPayload) {
  return apiRequest<{ term_id: string }>(`/calendar/terms`, {
    method: "POST",
    body: payload,
  });
}

export function detectDuplicates() {
  return apiRequest<{ flagged: number }>(
    `/content/documents/duplicates/detect`,
    {
      method: "POST",
    },
  );
}

export function setMergeFlag(duplicateId: string, mergeCandidate: boolean) {
  return apiRequest<{ status: string }>(
    `/content/documents/duplicates/${encodeURIComponent(duplicateId)}/merge-flag`,
    {
      method: "PATCH",
      body: { merge_candidate: mergeCandidate },
    },
  );
}

export function bulkUpdateDocuments(payload: {
  document_ids: string[];
  category_ids?: string[];
  tag_ids?: string[];
  archive?: boolean;
}) {
  return apiRequest<{ status: string }>(`/content/documents/bulk`, {
    method: "POST",
    body: payload,
  });
}

export function startContentUpload(payload: {
  document_id?: string;
  file_name: string;
  mime_type: string;
  total_chunks: number;
  chunk_size_bytes: number;
}) {
  return apiRequest<UploadSession>(`/content/uploads/start`, {
    method: "POST",
    body: payload,
  });
}

export async function uploadContentChunk(
  uploadID: string,
  chunkIndex: number,
  data: Uint8Array,
): Promise<void> {
  const normalized = new Uint8Array(data.length);
  normalized.set(data);
  const response = await fetch(
    `/api/v1/content/uploads/${encodeURIComponent(uploadID)}/chunks/${chunkIndex}`,
    {
      method: "PUT",
      credentials: "include",
      headers: {
        "Content-Type": "application/octet-stream",
      },
      body: normalized,
    },
  );
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw {
      status: response.status,
      message: payload?.error ?? "chunk upload failed",
    };
  }
}

export function completeContentUpload(
  uploadID: string,
  payload: {
    title: string;
    summary: string;
    difficulty: number;
    duration_minutes: number;
  },
) {
  return apiRequest<DocumentVersion>(
    `/content/uploads/${encodeURIComponent(uploadID)}/complete`,
    {
      method: "POST",
      body: payload,
    },
  );
}

export function getDocumentVersions(documentID: string) {
  return apiRequest<DocumentVersion[]>(
    `/content/documents/${encodeURIComponent(documentID)}/versions`,
  );
}

export function createDocumentShareLink(documentID: string, version?: number) {
  return apiRequest<{ token: string; expires_at: string }>(
    `/content/documents/${encodeURIComponent(documentID)}/share-links`,
    {
      method: "POST",
      body: version ? { version } : {},
    },
  );
}

export function createPlan(payload: {
  name: string;
  description: string;
  starts_on?: string;
  ends_on?: string;
}) {
  return apiRequest<{ plan_id: string }>(`/planning/plans`, {
    method: "POST",
    body: payload,
  });
}

export function createMilestone(
  planId: string,
  payload: {
    title: string;
    description: string;
    due_date?: string;
    sort_order: number;
  },
) {
  return apiRequest<{ milestone_id: string }>(
    `/planning/plans/${encodeURIComponent(planId)}/milestones`,
    {
      method: "POST",
      body: payload,
    },
  );
}

export function createTask(
  milestoneId: string,
  payload: {
    title: string;
    description: string;
    state: string;
    due_at?: string;
    estimated_minutes: number;
    sort_order: number;
  },
) {
  return apiRequest<{ task_id: string }>(
    `/planning/milestones/${encodeURIComponent(milestoneId)}/tasks`,
    {
      method: "POST",
      body: payload,
    },
  );
}

export function addDependency(taskId: string, dependsOnTaskId: string) {
  return apiRequest<{ status: string }>(
    `/planning/tasks/${encodeURIComponent(taskId)}/dependencies`,
    {
      method: "POST",
      body: { depends_on_task_id: dependsOnTaskId },
    },
  );
}

export function reorderTasks(milestoneId: string, orderedIds: string[]) {
  return apiRequest<{ status: string }>(
    `/planning/milestones/${encodeURIComponent(milestoneId)}/reorder-tasks`,
    {
      method: "PATCH",
      body: { ordered_ids: orderedIds },
    },
  );
}

export function bulkUpdateTasks(payload: {
  task_ids: string[];
  state?: string;
  actual_minutes?: number;
}) {
  return apiRequest<{ status: string }>(`/planning/tasks/bulk`, {
    method: "PATCH",
    body: payload,
  });
}

export function login(payload: {
  tenant_slug: string;
  username: string;
  password: string;
}) {
  return apiRequest<{ status: string }>(`/auth/login`, {
    method: "POST",
    body: payload,
  });
}

export function me() {
  return apiRequest<{ tenant_id: string; user_id: string; roles: string[] }>(
    `/auth/me`,
  );
}

export function logout() {
  return apiRequest<{ status: string }>(`/auth/logout`, { method: "POST" });
}
