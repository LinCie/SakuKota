import {
  api,
  apiResult,
  type ApiResult,
  type CommonApiErrorCode,
  type InvalidBodyErrorCode,
} from "~/shared/api";
import type { LoginResponse, Tokens, User } from "./types";

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
export type RefreshErrorCode =
  CommonApiErrorCode | InvalidBodyErrorCode | "invalid_refresh_token";
export type LogoutErrorCode = CommonApiErrorCode | InvalidBodyErrorCode;

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
): Promise<ApiResult<LoginResponse, LoginErrorCode>> {
  return apiResult<LoginResponse, LoginErrorCode>(
    api.post("auth/login", { json: body }),
  );
}

export function refresh(
  refreshToken: string,
): Promise<ApiResult<Tokens, RefreshErrorCode>> {
  return apiResult<Tokens, RefreshErrorCode>(
    api.post("auth/refresh", { json: { refresh_token: refreshToken } }),
  );
}

export function logout(
  refreshToken: string,
): Promise<ApiResult<void, LogoutErrorCode>> {
  return apiResult<void, LogoutErrorCode>(
    api.post("auth/logout", { json: { refresh_token: refreshToken } }),
  );
}
