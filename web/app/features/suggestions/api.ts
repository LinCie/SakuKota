import {
  apiGet,
  apiPost,
  type AdminWriteErrorCode,
  type AuthApiErrorCode,
  type CommonApiErrorCode,
  type InvalidBodyErrorCode,
  unwrap,
} from "~/shared/api";
import type { PlaceRequest } from "~/features/places/api";
import type { Suggestion, SuggestionStatus } from "./types";

// -----------------------------------------------------------------------------
// Request params / error codes (service-layer concerns, not domain shapes).
// Services resolve to the response data and throw an `ApiError` carrying one
// of these codes when the backend rejects the request.
// -----------------------------------------------------------------------------

export type CreateSuggestionErrorCode =
  | CommonApiErrorCode
  | InvalidBodyErrorCode
  | "UNAUTHORIZED"
  | "VALIDATION_ERROR"
  | "TOO_MANY_PENDING_SUGGESTIONS";
export type GetMySuggestionsErrorCode = CommonApiErrorCode | "UNAUTHORIZED";
export type GetSuggestionsErrorCode =
  CommonApiErrorCode | AuthApiErrorCode | "INVALID_STATUS";
export type GetSuggestionErrorCode =
  | CommonApiErrorCode
  | AuthApiErrorCode
  | "INVALID_SUGGESTION_ID"
  | "SUGGESTION_NOT_FOUND";
export type ReviewSuggestionErrorCode =
  | AdminWriteErrorCode
  | "INVALID_SUGGESTION_ID"
  | "VALIDATION_ERROR"
  | "SUGGESTION_NOT_FOUND"
  | "ALREADY_REVIEWED";

export type GetSuggestionsParams = {
  /** Keeps only suggestions in this status, matching the `status` query param. */
  status?: SuggestionStatus;
};

/** A place as the user knows it, plus a note to the reviewer. */
export type CreateSuggestionRequest = PlaceRequest & {
  /** Trimmed by the backend; a blank note is stored as null. */
  note?: string | null;
};

/** The place the admin creates from the suggestion. */
export type ApproveSuggestionRequest = PlaceRequest;

export type RejectSuggestionRequest = {
  /** Trimmed by the backend; a blank reason is stored as null. */
  reason?: string | null;
};

// -----------------------------------------------------------------------------
// Services (see internal/router/router.go)
// -----------------------------------------------------------------------------

/** Signed-in users only. Stored as pending review. */
export async function createSuggestion(
  body: CreateSuggestionRequest,
): Promise<Suggestion> {
  return unwrap(
    await apiPost<Suggestion, CreateSuggestionErrorCode>("suggestions", {
      json: body,
    }),
  );
}

/** Signed-in users only. The user's own suggestions, newest first. */
export async function getMySuggestions(): Promise<Suggestion[]> {
  return unwrap(
    await apiGet<Suggestion[], GetMySuggestionsErrorCode>("suggestions/mine"),
  );
}

/** Admin only. Oldest first. */
export async function getSuggestions(
  params: GetSuggestionsParams = {},
): Promise<Suggestion[]> {
  const searchParams = new URLSearchParams();
  if (params.status) searchParams.set("status", params.status);

  return unwrap(
    await apiGet<Suggestion[], GetSuggestionsErrorCode>("suggestions", {
      searchParams,
    }),
  );
}

/** Admin only. */
export async function getSuggestion(id: number): Promise<Suggestion> {
  return unwrap(
    await apiGet<Suggestion, GetSuggestionErrorCode>(`suggestions/${id}`),
  );
}

/** Admin only. Creates `body` as a place and marks the suggestion approved. */
export async function approveSuggestion(
  id: number,
  body: ApproveSuggestionRequest,
): Promise<Suggestion> {
  return unwrap(
    await apiPost<Suggestion, ReviewSuggestionErrorCode>(
      `suggestions/${id}/approve`,
      { json: body },
    ),
  );
}

/** Admin only. */
export async function rejectSuggestion(
  id: number,
  body: RejectSuggestionRequest,
): Promise<Suggestion> {
  return unwrap(
    await apiPost<Suggestion, ReviewSuggestionErrorCode>(
      `suggestions/${id}/reject`,
      { json: body },
    ),
  );
}
