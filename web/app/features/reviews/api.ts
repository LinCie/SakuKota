import {
  apiGet,
  apiPost,
  type CommonApiErrorCode,
  type InvalidBodyErrorCode,
  unwrap,
} from "~/shared/api";
import type { Review } from "./types";

// -----------------------------------------------------------------------------
// Request params / error codes (service-layer concerns, not domain shapes).
// Services resolve to the response data and throw an `ApiError` carrying one
// of these codes when the backend rejects the request.
// -----------------------------------------------------------------------------

export type GetReviewsErrorCode =
  CommonApiErrorCode | "invalid_place_id" | "place_not_found";
export type CreateReviewErrorCode =
  | CommonApiErrorCode
  | InvalidBodyErrorCode
  | "UNAUTHORIZED"
  | "invalid_place_id"
  | "validation_error"
  | "place_not_found"
  | "already_reviewed";

export type CreateReviewRequest = {
  /** Stars, from 1 to 5. */
  rating: number;
  /** Trimmed by the backend; a blank comment is stored as null. */
  comment?: string | null;
};

// -----------------------------------------------------------------------------
// Services (see internal/router/router.go)
// -----------------------------------------------------------------------------

/** The place's reviews, newest first. */
export async function getPlaceReviews(placeId: number): Promise<Review[]> {
  return unwrap(
    await apiGet<Review[], GetReviewsErrorCode>(`places/${placeId}/reviews`),
  );
}

/** Signed-in users only; each user reviews a place once. */
export async function createPlaceReview(
  placeId: number,
  body: CreateReviewRequest,
): Promise<Review> {
  return unwrap(
    await apiPost<Review, CreateReviewErrorCode>(`places/${placeId}/reviews`, {
      json: body,
    }),
  );
}
