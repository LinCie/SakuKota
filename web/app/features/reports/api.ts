import { apiGet, apiPost, unwrap } from "~/shared/api";
import type { Report, ReportDetail, ReportReason, ReportStatus } from "./types";

// Every function here calls one backend route (see internal/router/router.go),
// returns the response data, and throws an `ApiError` when the backend rejects
// the request.

export type CreateReportRequest = {
  reason: ReportReason;
  /** Trimmed by the backend; a blank note is stored as null. */
  note?: string | null;
};

export type GetReportsParams = {
  /** Keeps only reports in this status, matching the `status` query param. */
  status?: ReportStatus;
};

/**
 * Signed-in users only. Reports the place, or its review when `reviewId` is
 * set. A user has one pending report per content and at most 5 pending.
 */
export async function createReport(
  placeId: number,
  reviewId: number | null,
  body: CreateReportRequest,
): Promise<Report> {
  const path =
    reviewId === null
      ? `places/${placeId}/reports`
      : `places/${placeId}/reviews/${reviewId}/reports`;
  return unwrap(await apiPost<Report>(path, { json: body }));
}

/** Admin only. Oldest first. */
export async function getReports(
  params: GetReportsParams = {},
): Promise<Report[]> {
  const searchParams = new URLSearchParams();
  if (params.status) searchParams.set("status", params.status);

  return unwrap(await apiGet<Report[]>("reports", { searchParams }));
}

/** Admin only. The report with its content as it is now. */
export async function getReport(id: number): Promise<ReportDetail> {
  return unwrap(await apiGet<ReportDetail>(`reports/${id}`));
}

/** Admin only. Deletes the reported content and resolves its pending reports. */
export async function removeReportedContent(id: number): Promise<Report> {
  return unwrap(await apiPost<Report>(`reports/${id}/remove`));
}

/** Admin only. Keeps the content and resolves this report only. */
export async function dismissReport(id: number): Promise<Report> {
  return unwrap(await apiPost<Report>(`reports/${id}/dismiss`));
}
