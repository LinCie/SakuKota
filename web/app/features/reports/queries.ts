import {
  queryOptions,
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";

import { placeKeys } from "~/features/places/queries";
import { reviewKeys } from "~/features/reviews/queries";

import {
  createReport,
  dismissReport,
  getReport,
  getReports,
  removeReportedContent,
  type CreateReportRequest,
  type GetReportsParams,
} from "./api";

export const reportKeys = {
  all: ["reports"] as const,
  list: (params: GetReportsParams) => ["reports", "list", params] as const,
  detail: (id: number) => ["reports", "detail", id] as const,
};

// Queries. The options are shared by route loaders (`query`) and the hooks
// below, which read the data the loader put in the cache.

export function reportsQuery(params: GetReportsParams = {}) {
  return queryOptions({
    queryKey: reportKeys.list(params),
    queryFn: () => getReports(params),
  });
}

export function reportQuery(id: number) {
  return queryOptions({
    queryKey: reportKeys.detail(id),
    queryFn: () => getReport(id),
  });
}

export function useReports(params: GetReportsParams = {}) {
  return useSuspenseQuery(reportsQuery(params));
}

export function useReport(id: number) {
  return useSuspenseQuery(reportQuery(id));
}

// Mutations. Each resolves once the affected queries refetched.

/** Reports the place, or its review when `reviewId` is set. */
export function useCreateReport(placeId: number, reviewId: number | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateReportRequest) =>
      createReport(placeId, reviewId, body),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: reportKeys.all }),
  });
}

/** Admin only. Removing deletes a place or a review, so those refetch too. */
export function useRemoveReportedContent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: removeReportedContent,
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: reportKeys.all }),
        queryClient.invalidateQueries({ queryKey: placeKeys.all }),
        queryClient.invalidateQueries({ queryKey: reviewKeys.all }),
      ]),
  });
}

/** Admin only. */
export function useDismissReport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: dismissReport,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: reportKeys.all }),
  });
}
