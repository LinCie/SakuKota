// -----------------------------------------------------------------------------
// Types (see internal/module/review/dto.go)
// -----------------------------------------------------------------------------

/** The user who wrote a review. */
export type ReviewAuthor = {
  id: number;
  name: string;
};

export type Review = {
  id: number;
  place_id: number;
  /** Stars, from 1 to 5. */
  rating: number;
  /** Null when the review is only a rating. */
  comment: string | null;
  created_at: string;
  user: ReviewAuthor;
};
