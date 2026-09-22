export class APIError extends Error {
  constructor(public status: number, message: string) { super(message); }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const unsafe = init.method && !["GET", "HEAD", "OPTIONS"].includes(init.method.toUpperCase());
  const csrf = typeof document === "undefined" ? "" : document.cookie.split("; ").find(v => v.startsWith("erp_csrf="))?.split("=")[1] ?? "";
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(unsafe && csrf ? {"X-CSRF-Token": decodeURIComponent(csrf)} : {}), ...init.headers },
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new APIError(response.status, body.message ?? "Permintaan gagal");
  return body as T;
}
