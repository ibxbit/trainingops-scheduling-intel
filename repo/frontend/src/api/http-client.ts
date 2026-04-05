export type ApiRequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  signal?: AbortSignal;
  headers?: Record<string, string>;
};

export type ApiError = {
  status: number;
  message: string;
  reason?: unknown;
  alternatives?: unknown;
};

const API_BASE = "/api/v1";

export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const method = options.method ?? "GET";

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    credentials: "include",
    signal: options.signal,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers ?? {}),
    },
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });

  const payload = await response.json().catch(() => ({}));

  if (!response.ok) {
    const error: ApiError = {
      status: response.status,
      message: payload?.error ?? "Request failed",
      reason: payload?.reason,
      alternatives: payload?.alternatives,
    };
    throw error;
  }

  return payload.data as T;
}
