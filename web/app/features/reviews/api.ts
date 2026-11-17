import { apiGet, apiPost, unwrap } from "~/shared/api";
import type { Review } from "./types";

// Every function here calls one backend route (see internal/router/router.go),
// returns the response data, and throws an `ApiError` when the backend rejects
// the request.

export type CreateReviewRequest = {
  /** Stars, from 1 to 5. */
  rating: number;
  /** Trimmed by the backend; a blank comment is stored as null. */
  comment?: string | null;
};

/** The place's reviews, newest first. */
export async function getPlaceReviews(placeId: number): Promise<Review[]> {
  return unwrap(await apiGet<Review[]>(`places/${placeId}/reviews`));
}

/** Signed-in users only; each user reviews a place once. */
export async function createPlaceReview(
  placeId: number,
  body: CreateReviewRequest,
): Promise<Review> {
  return unwrap(
    await apiPost<Review>(`places/${placeId}/reviews`, { json: body }),
  );
}
