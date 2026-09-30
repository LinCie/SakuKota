import {
  api,
  apiResult,
  type ApiResult,
  type CommonApiErrorCode,
  type InvalidBodyErrorCode,
} from "~/shared/api";
import type { Session, User } from "./types";

// -----------------------------------------------------------------------------
// Request bodies / error codes (service-layer concerns, not domain shapes)
// -----------------------------------------------------------------------------

export type RegisterErrorCode =
  | CommonApiErrorCode
  | InvalidBodyErrorCode
  | "validation_error"
  | "email_taken";
export type LoginErrorCode =
  CommonApiErrorCode | InvalidBodyErrorCode | "invalid_credentials";
export type RefreshErrorCode = CommonApiErrorCode | "invalid_refresh_token";
export type LogoutErrorCode = CommonApiErrorCode;

export type RegisterRequest = {
  name: string;
  email: string;
  password: string;
};

export type LoginRequest = {
  email: string;
  password: string;
};

// -----------------------------------------------------------------------------
// Services (see internal/module/auth/route.go)
// -----------------------------------------------------------------------------

export function register(
  body: RegisterRequest,
): Promise<ApiResult<User, RegisterErrorCode>> {
  return apiResult<User, RegisterErrorCode>(
    api.post("auth/register", { json: body }),
  );
}

export function login(
  body: LoginRequest,
): Promise<ApiResult<Session, LoginErrorCode>> {
  return apiResult<Session, LoginErrorCode>(
    api.post("auth/login", { json: body }),
  );
}

/**
 * Exchanges the HttpOnly refresh cookie for a new session. Prefer
 * `refreshSession` (see ./session), which serializes refreshes.
 */
export function refresh(): Promise<ApiResult<Session, RefreshErrorCode>> {
  return apiResult<Session, RefreshErrorCode>(api.post("auth/refresh"));
}

/** Revokes the refresh cookie sent with the request and clears it. */
export function logout(): Promise<ApiResult<void, LogoutErrorCode>> {
  return apiResult<void, LogoutErrorCode>(api.post("auth/logout"));
}
