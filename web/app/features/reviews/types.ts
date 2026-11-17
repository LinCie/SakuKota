// Types (see internal/module/review/dto.go)

export type Review = {
  id: number;
  place_id: number;
  /** Stars, from 1 to 5. */
  rating: number;
  /** Null when the review is only a rating. */
  comment: string | null;
  created_at: string;
  /** The user who wrote the review. */
  user: { id: number; name: string };
};
