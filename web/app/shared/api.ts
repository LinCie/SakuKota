import ky from "ky";

// The rantau-go backend has no authentication yet, so this client only
// centralizes base URL configuration, retry behavior, and response
// normalization. If the backend later adds session/token authentication,
// the refresh-and-retry-once logic should be added here (e.g. via an
// `afterResponse` hook) so it stays out of individual domain services.
const apiBaseUrl = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

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
