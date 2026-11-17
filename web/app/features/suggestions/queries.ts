import {
  queryOptions,
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";

import type { PlaceRequest } from "~/features/places/api";
import { placeKeys } from "~/features/places/queries";

import {
  approveSuggestion,
  createSuggestion,
  getMySuggestions,
  getSuggestion,
  getSuggestions,
  rejectSuggestion,
  type GetSuggestionsParams,
  type RejectSuggestionRequest,
} from "./api";

export const suggestionKeys = {
  all: ["suggestions"] as const,
  list: (params: GetSuggestionsParams) =>
    ["suggestions", "list", params] as const,
  detail: (id: number) => ["suggestions", "detail", id] as const,
  /** Keyed by user, so one user's suggestions never show for the next. */
  mine: (userId: number) => ["suggestions", "mine", userId] as const,
};

// Queries. The options are shared by route loaders (`query`) and the
// hooks below, which read the data the loader put in the cache.

export function mySuggestionsQuery(userId: number) {
  return queryOptions({
    queryKey: suggestionKeys.mine(userId),
    queryFn: getMySuggestions,
  });
}

export function suggestionsQuery(params: GetSuggestionsParams = {}) {
  return queryOptions({
    queryKey: suggestionKeys.list(params),
    queryFn: () => getSuggestions(params),
  });
}

export function suggestionQuery(id: number) {
  return queryOptions({
    queryKey: suggestionKeys.detail(id),
    queryFn: () => getSuggestion(id),
  });
}

export function useMySuggestions(userId: number) {
  return useSuspenseQuery(mySuggestionsQuery(userId));
}

export function useSuggestions(params: GetSuggestionsParams = {}) {
  return useSuspenseQuery(suggestionsQuery(params));
}

export function useSuggestion(id: number) {
  return useSuspenseQuery(suggestionQuery(id));
}

// Mutations. Each resolves once the affected queries refetched.

export function useCreateSuggestion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createSuggestion,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: suggestionKeys.all }),
  });
}

/** Admin only. Approving adds a place, so the places refetch too. */
export function useApproveSuggestion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: PlaceRequest }) =>
      approveSuggestion(id, body),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: suggestionKeys.all }),
        queryClient.invalidateQueries({ queryKey: placeKeys.all }),
      ]),
  });
}

/** Admin only. */
export function useRejectSuggestion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: RejectSuggestionRequest }) =>
      rejectSuggestion(id, body),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: suggestionKeys.all }),
  });
}
