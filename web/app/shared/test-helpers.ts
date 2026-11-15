import { vi } from "vitest";

// Helpers for tests that run the real API client and session against a fake
// backend: only `fetch` is replaced.

type Handler = (request: Request) => Response | Promise<Response>;

/** Replaces `fetch` with `handler`; `requests` collects what was sent, in order. */
export function fakeBackend(handler: Handler) {
  const requests: Request[] = [];
  vi.stubGlobal("fetch", (input: RequestInfo | URL, init?: RequestInit) => {
    const request = new Request(input, init);
    requests.push(request);
    return Promise.resolve(handler(request));
  });
  return {
    requests,
    /** Requests made to `path` (below the API's base URL). */
    to: (path: string) =>
      requests.filter((request) => pathOf(request) === `/api/${path}`),
  };
}

function pathOf(request: Request) {
  return new URL(request.url).pathname;
}

/** The path of `request` below the API's base URL, e.g. "auth/refresh". */
export function routeOf(request: Request) {
  return pathOf(request).replace(/^\/api\//, "");
}

export function okResponse(data: unknown, status = 200) {
  return Response.json({ data }, { status });
}

export function errorResponse(status: number, code: string, message = code) {
  return Response.json({ code, message }, { status });
}

/** A token with the given claims; only the payload matters to the frontend. */
export function fakeJwt(claims: unknown) {
  const encode = (value: unknown) =>
    Buffer.from(JSON.stringify(value)).toString("base64url");
  return `${encode({ alg: "none" })}.${encode(claims)}.signature`;
}

export function sessionOf(accessToken: string) {
  return {
    user: {
      id: 1,
      name: "Siti",
      email: "siti@example.com",
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    },
    access_token: accessToken,
  };
}

/** A promise settled from outside, to hold a response while the test acts. */
export function gate<T = void>() {
  let open!: (value: T) => void;
  const promise = new Promise<T>((resolve) => {
    open = resolve;
  });
  return { promise, open };
}

/** The app's API client and session, freshly loaded and wired as main.tsx does. */
export async function loadApp() {
  vi.resetModules();
  const api = await import("./api");
  const session = await import("~/features/auth/session");
  api.setAuthHandlers({
    getAccessToken: session.getAccessToken,
    refresh: session.refreshSession,
  });
  return { api, session };
}
