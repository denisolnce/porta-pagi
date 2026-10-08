// Thin fetch wrapper. The device token lives in localStorage and rides as a
// bearer header; the server also keeps the same token in a cookie it sets
// on /api/me (HttpOnly, SameSite=Strict — another site cannot send it),
// which is what survives Safari forgetting this drawer after a quiet week.
// A fetch here is same-origin, so the cookie rides on its own; the header
// goes only while the drawer still has the token. Raw fetches elsewhere
// (photo.ts, the export) take authHeaders() for the same reason.
export const API = "/api";
const KEY = "potok.token";

export function getToken(): string | null {
  try { return localStorage.getItem(KEY); } catch { return null; }
}
export function setToken(t: string | null) {
  try { t ? localStorage.setItem(KEY, t) : localStorage.removeItem(KEY); } catch { /* private mode */ }
}
export function authHeaders(): Record<string, string> {
  const tok = getToken();
  return tok ? { Authorization: "Bearer " + tok } : {};
}

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message); }
}

export async function api<T = any>(path: string, opts: { method?: string; body?: any } = {}): Promise<T> {
  const headers: Record<string, string> = authHeaders();
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(API + path, {
    method: opts.method || "GET",
    headers,
    body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
  });
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  let data: any = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = null; }
  if (res.status === 401 && path !== "/me" && !path.startsWith("/join") && !path.startsWith("/bootstrap") && !path.startsWith("/demo/")) {
    // The session is dead — a phone a steward removed, or the demo's hourly
    // reset. Fall to the gate instead of to a page of errors. /me is the
    // call that asks whether there is a session at all: App answers its
    // 401 with the gate, and a reload here would ask again forever.
    setToken(null); location.reload();
  }
  if (!res.ok) throw new ApiError(res.status, (data && data.error) || res.statusText);
  return data as T;
}

export type House = { id: number; name: string; crest: string; color: string; kind: "house" | "common"; is_steward: number; parcels: string[]; homes: string[] };
export type Me = { id: number; name: string; crest: string; color: string; kind: string; is_steward: number; device_id: number };
