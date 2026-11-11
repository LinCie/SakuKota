import type { ApiResult } from "~/shared/api";
import { refresh, type RefreshErrorCode } from "./api";
import type { Session } from "./types";

// The session lives in memory only, never in web storage, so the access token
// is gone on reload; `refreshSession` restores it from the HttpOnly refresh
// cookie at startup. Components re-render through `useSession` whenever it
// changes.
const listeners = new Set<() => void>();
let session: Session | null = null;

function writeSession(next: Session | null) {
  session = next;
  for (const listener of listeners) listener();
}

export function subscribeSession(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getSession() {
  return session;
}

export function getAccessToken() {
  return session?.access_token;
}

export function saveSession(next: Session) {
  writeSession(next);
}

export function clearSession() {
  writeSession(null);
}

/**
 * Whether the access token carries the `admin` role. Only used to decide what
 * to show: the backend enforces the role on every admin endpoint.
 */
export function isAdmin(current: Session | null): boolean {
  if (!current) return false;
  try {
    const payload = current.access_token.split(".")[1] ?? "";
    const claims: unknown = JSON.parse(
      atob(payload.replace(/-/g, "+").replace(/_/g, "/")),
    );
    return (
      typeof claims === "object" &&
      claims !== null &&
      "role" in claims &&
      claims.role === "admin"
    );
  } catch {
    return false;
  }
}

// -----------------------------------------------------------------------------
// Refresh
// -----------------------------------------------------------------------------

// The backend rotates refresh tokens (each one is single-use), so refreshes
// must never overlap: within a tab, callers share one in-flight refresh;
// across tabs, a Web Lock makes them take turns, so each sends the cookie the
// previous one set.
let refreshing: Promise<boolean> | null = null;

/**
 * Exchanges the HttpOnly refresh cookie for a new session, both after an
 * access token expires and on page load. Resolves to whether the user is
 * signed in afterwards.
 */
export function refreshSession(): Promise<boolean> {
  refreshing ??= withRefreshLock(doRefreshSession).finally(() => {
    refreshing = null;
  });
  return refreshing;
}

let restoring: Promise<boolean> | null = null;

/**
 * Restores the session from the refresh cookie once per page load. Startup
 * begins it without waiting, so the first render isn't blocked; route guards
 * await it before reading the session.
 */
export function restoreSession(): Promise<boolean> {
  restoring ??= refreshSession();
  return restoring;
}

async function withRefreshLock(
  callback: () => Promise<boolean>,
): Promise<boolean> {
  // Web Locks need a secure context (HTTPS or localhost).
  return "locks" in navigator
    ? navigator.locks.request("rantau:refresh", callback)
    : callback();
}

async function doRefreshSession(): Promise<boolean> {
  const before = session;

  let result: ApiResult<Session, RefreshErrorCode>;
  try {
    result = await refresh();
  } catch {
    // Network/protocol failure: keep the session so a later call can retry.
    return false;
  }

  // The user signed out (or in) while the refresh was in flight.
  if (session !== before) return false;

  if (!result.ok) {
    // 401 means the cookie is missing, expired, revoked, or already used.
    if (result.status === 401) clearSession();
    return false;
  }

  saveSession(result.data);
  return true;
}
