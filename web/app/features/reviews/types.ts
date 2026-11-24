import type { Media } from "~/shared/image-upload";

// Types (see internal/module/reviews/dto.go)

export type Review = {
  id: number;
  place_id: number;
  place_name: string;
  /** Stars, from 1 to 5. */
  rating: number;
  /** Null when the review is only a rating. */
  comment: string | null;
  created_at: string;
  /** The user who wrote the review, with their profile photo if any. */
  user: { id: number; name: string; avatar_url: string | null };
  /** In display order; empty when the review has none. */
  medias: Media[];
};
