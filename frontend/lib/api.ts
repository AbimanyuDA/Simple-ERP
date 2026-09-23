export class APIError extends Error {
  constructor(public status: number, message: string) { super(message); }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  return request<T>(path, init, true);
}

async function request<T>(path: string, init: RequestInit, allowRefresh: boolean): Promise<T> {
  const unsafe = init.method && !["GET", "HEAD", "OPTIONS"].includes(init.method.toUpperCase());
  const csrf = typeof document === "undefined" ? "" : document.cookie.split("; ").find(v => v.startsWith("erp_csrf="))?.split("=")[1] ?? "";
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(unsafe ? { "Idempotency-Key": crypto.randomUUID() } : {}),
      ...(unsafe && csrf ? {"X-CSRF-Token": decodeURIComponent(csrf)} : {}),
      ...init.headers,
    },
  });
  if (response.status === 401 && allowRefresh && !path.startsWith("/auth/")) {
    const refreshed = await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" } });
    if (refreshed.ok) return request<T>(path, init, false);
  }
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new APIError(response.status, body.message ?? "Permintaan gagal");
  return body as T;
}
