import type { GeoPoint } from "~/features/places/types";

// Types (see internal/module/suggestion/dto.go)

/** A suggestion is reviewed once: it leaves pending for approved or rejected. */
export const SUGGESTION_STATUSES = ["pending", "approved", "rejected"] as const;
export type SuggestionStatus = (typeof SUGGESTION_STATUSES)[number];

export type Suggestion = {
  id: number;
  sub_district_id: number;
  category_id: number;
  name: string;
  address: string | null;
  location: GeoPoint | null;
  operating_hours: string | null;
  estimated_price_range: string | null;
  short_description: string | null;
  /** Values of the category's custom fields, keyed by `CategoryField.key`. */
  attributes: Record<string, unknown>;
  /** The user's message to the reviewer. */
  note: string | null;
  status: SuggestionStatus;
  /** Null unless the suggestion was rejected with a reason. */
  rejection_reason: string | null;
  /** The place the suggestion was approved as; null if it has none, or it was deleted since. */
  place_id: number | null;
  reviewed_at: string | null;
  created_at: string;
  /** The user who suggested the place. */
  user: { id: number; name: string };
};
