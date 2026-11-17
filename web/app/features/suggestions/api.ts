import { apiGet, apiPost, unwrap } from "~/shared/api";
import type { PlaceRequest } from "~/features/places/api";
import type { Suggestion, SuggestionStatus } from "./types";

// Every function here calls one backend route (see internal/router/router.go),
// returns the response data, and throws an `ApiError` when the backend rejects
// the request.

export type GetSuggestionsParams = {
  /** Keeps only suggestions in this status, matching the `status` query param. */
  status?: SuggestionStatus;
};

/** A place as the user knows it, plus a note to the reviewer. */
export type CreateSuggestionRequest = PlaceRequest & {
  /** Trimmed by the backend; a blank note is stored as null. */
  note?: string | null;
};

export type RejectSuggestionRequest = {
  /** Trimmed by the backend; a blank reason is stored as null. */
  reason?: string | null;
};

/** Signed-in users only. Stored as pending review. */
export async function createSuggestion(
  body: CreateSuggestionRequest,
): Promise<Suggestion> {
  return unwrap(await apiPost<Suggestion>("suggestions", { json: body }));
}

/** Signed-in users only. The user's own suggestions, newest first. */
export async function getMySuggestions(): Promise<Suggestion[]> {
  return unwrap(await apiGet<Suggestion[]>("suggestions/mine"));
}

/** Admin only. Oldest first. */
export async function getSuggestions(
  params: GetSuggestionsParams = {},
): Promise<Suggestion[]> {
  const searchParams = new URLSearchParams();
  if (params.status) searchParams.set("status", params.status);

  return unwrap(await apiGet<Suggestion[]>("suggestions", { searchParams }));
}

/** Admin only. */
export async function getSuggestion(id: number): Promise<Suggestion> {
  return unwrap(await apiGet<Suggestion>(`suggestions/${id}`));
}

/** Admin only. Creates `body` as a place and marks the suggestion approved. */
export async function approveSuggestion(
  id: number,
  body: PlaceRequest,
): Promise<Suggestion> {
  return unwrap(
    await apiPost<Suggestion>(`suggestions/${id}/approve`, { json: body }),
  );
}

/** Admin only. */
export async function rejectSuggestion(
  id: number,
  body: RejectSuggestionRequest,
): Promise<Suggestion> {
  return unwrap(
    await apiPost<Suggestion>(`suggestions/${id}/reject`, { json: body }),
  );
}
