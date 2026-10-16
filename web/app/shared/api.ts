import ky, { type Options } from "ky";

// This client centralizes base URL configuration, retry behavior, auth, and
// response normalization. The signed-in user's access token is attached to
// every request, and a 401 triggers one token refresh and retry, so none of
// this leaks into domain services.
//
// The API is served from the app's own origin (the Vite dev server proxies
// `/api`, see vite.config.ts), which keeps the HttpOnly refresh cookie
// first-party. The trailing slash makes `baseUrl` resolve relative inputs
// under the prefix rather than replacing its last segment.
const apiBaseUrl = `${(import.meta.env.VITE_API_URL || "/api").replace(/\/+$/, "")}/`;

/**
 * How the client reaches the session, supplied by the auth feature at startup
 * (see main.tsx) so that shared code never depends on a feature.
 */
export type AuthHandlers = {
  /** The current access token, if signed in. */
  getAccessToken: () => string | undefined;
  /** Renews the access token after a 401; resolves to whether it did. */
  refresh: () => Promise<boolean>;
};

let auth: AuthHandlers | undefined;

export function setAuthHandlers(handlers: AuthHandlers) {
  auth = handlers;
}

function setAuthHeader({ request }: { request: Request }) {
  const token = auth?.getAccessToken();
  if (token) request.headers.set("Authorization", `Bearer ${token}`);
}

// `throwHttpErrors: false` means non-2xx responses are returned as regular
// `Response` objects instead of being thrown, so `ky`'s built-in retry only
// kicks in for genuine transport failures (network errors), not for normal
// backend application errors. Those are normalized into `ApiResult` below.
export const api = ky.create({
  baseUrl: apiBaseUrl,
  throwHttpErrors: false,
  retry: {
    limit: 2,
  },
  hooks: {
    beforeRequest: [setAuthHeader],
    // Retries reuse the original request, so re-attach the current (possibly
    // refreshed) access token.
    beforeRetry: [setAuthHeader],
    afterResponse: [
      async ({ request, response, retryCount }) => {
        // Auth endpoints answer 401 for bad credentials or refresh tokens;
        // refreshing on those would recurse into the refresh call itself.
        const isAuthRoute = new URL(request.url).pathname.includes("/auth/");
        if (retryCount > 0 || response.status !== 401 || isAuthRoute) return;
        // Anonymous requests have no expired token to renew.
        if (!auth?.getAccessToken()) return;

        // Forced retries bypass `retry.methods`, so this also covers POSTs.
        if (await auth.refresh()) return ky.retry();
      },
    ],
  },
});

// -----------------------------------------------------------------------------
// Backend response envelopes
// -----------------------------------------------------------------------------

/** The backend's `{ "data": T }` success envelope (see internal/httpx.RespondJSON). */
export type ApiResponse<T> = {
  data: T;
};

/** The backend's `{ "code": string, "message": string }` error envelope (see internal/httpx.RespondError). */
export type ApiErrorResponse<C extends string = string> = {
  code: C;
  message: string;
};

/** Error code shared by every backend endpoint (unhandled server-side failures). */
export type CommonApiErrorCode = "internal_error";

/** Error code for endpoints that reject an undecodable request body. */
export type InvalidBodyErrorCode = "invalid_body";

/** Error codes for endpoints behind sign-in and a role check (see internal/middleware). */
export type AuthApiErrorCode = "UNAUTHORIZED" | "FORBIDDEN";

/** Error codes shared by admin-only writes: sign-in and role checks, plus invalid bodies. */
export type AdminWriteErrorCode =
  CommonApiErrorCode | AuthApiErrorCode | InvalidBodyErrorCode;

/**
 * Normalized outcome of a backend call. `ok: true` carries the typed
 * response data; `ok: false` carries the HTTP status and the typed backend
 * error payload. UI code should branch on `ok` rather than catching thrown
 * errors for expected backend application errors.
 */
export type ApiResult<T, C extends string = string> =
  | {
      ok: true;
      data: T;
    }
  | {
      ok: false;
      status: number;
      error: ApiErrorResponse<C>;
    };

/**
 * Thrown when an HTTP response exists but does not match the backend's
 * documented envelope (malformed JSON, or a success/error body missing the
 * expected shape). Distinct from network/transport failures, which surface
 * as whatever error `ky`/`fetch` throws.
 */
export class ApiProtocolError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "ApiProtocolError";
  }
}

function isApiResponse<T>(body: unknown): body is ApiResponse<T> {
  return typeof body === "object" && body !== null && "data" in body;
}

function isApiErrorResponse(body: unknown): body is ApiErrorResponse {
  return (
    typeof body === "object" &&
    body !== null &&
    typeof (body as Record<string, unknown>).code === "string" &&
    typeof (body as Record<string, unknown>).message === "string"
  );
}

/**
 * Awaits a `ky` request and normalizes it into an `ApiResult`, validating
 * the backend's success/error envelope along the way. This is the only
 * place response parsing should happen; domain services should not
 * re-implement it.
 */
export async function apiResult<T, C extends string = string>(
  responsePromise: Promise<Response>,
): Promise<ApiResult<T, C>> {
  const response = await responsePromise;

  // 204 No Content has no envelope (e.g. POST /auth/logout).
  if (response.status === 204) {
    return {
      ok: true,
      data: undefined as T,
    };
  }

  let body: unknown;
  try {
    body = await response.json();
  } catch (error) {
    throw new ApiProtocolError(
      `Failed to parse response body as JSON (status ${response.status})`,
      { cause: error },
    );
  }

  if (response.ok) {
    if (!isApiResponse<T>(body)) {
      throw new ApiProtocolError(
        "Response does not match the expected API success envelope",
      );
    }

    return {
      ok: true,
      data: body.data,
    };
  }

  if (!isApiErrorResponse(body)) {
    throw new ApiProtocolError(
      "Response does not match the expected API error envelope",
    );
  }

  return {
    ok: false,
    status: response.status,
    error: body as ApiErrorResponse<C>,
  };
}

/** GET `path` and normalize the response into an `ApiResult`. */
export function apiGet<T, C extends string = string>(
  path: string,
  options?: Options,
): Promise<ApiResult<T, C>> {
  return apiResult<T, C>(api.get(path, options));
}

/** POST to `path` and normalize the response into an `ApiResult`. */
export function apiPost<T, C extends string = string>(
  path: string,
  options?: Options,
): Promise<ApiResult<T, C>> {
  return apiResult<T, C>(api.post(path, options));
}
