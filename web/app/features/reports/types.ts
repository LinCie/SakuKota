import type { Category } from "~/features/places/types";
import type { Media } from "~/shared/image-upload";

// Types (see internal/module/reports/dto.go)

export const REPORT_REASONS = ["spam", "offensive", "fake", "other"] as const;
export type ReportReason = (typeof REPORT_REASONS)[number];

/** A report is resolved once: it leaves pending for removed or dismissed. */
export const REPORT_STATUSES = ["pending", "removed", "dismissed"] as const;
export type ReportStatus = (typeof REPORT_STATUSES)[number];

export const REASON_LABELS: Record<ReportReason, string> = {
  spam: "Spam atau iklan",
  offensive: "Kasar atau menyinggung",
  fake: "Palsu atau menyesatkan",
  other: "Lainnya",
};

export type Report = {
  id: number;
  /** The reported place, or the place of the reported review. It may be deleted since. */
  place_id: number;
  /** The reported review; null when the place itself is reported. */
  review_id: number | null;
  reason: ReportReason;
  note: string | null;
  status: ReportStatus;
  resolved_at: string | null;
  created_at: string;
  /** The user who filed the report. */
  user: { id: number; name: string };
};

/** The part of a reported place an admin judges it by. */
export type ReportedPlace = {
  id: number;
  name: string;
  address: string | null;
  short_description: string | null;
  category: Category;
  medias: Media[];
};

export type ReportedReview = {
  id: number;
  /** Stars, from 1 to 5. */
  rating: number;
  comment: string | null;
  created_at: string;
  /** The user who wrote the review. */
  user: { id: number; name: string };
  medias: Media[];
};

/** A report with its content as it is now. */
export type ReportDetail = Report & {
  /** The place, or the reviewed place; null once deleted. */
  place: ReportedPlace | null;
  /** The reported review; null for a place report or once deleted. */
  review: ReportedReview | null;
};
