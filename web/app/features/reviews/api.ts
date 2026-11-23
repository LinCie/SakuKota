import { apiDelete, apiGet, apiPost, unwrap } from "~/shared/api";
import type { MediaUpload } from "~/shared/image-upload";
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

// Media routes act on the signed-in user's own review of the place.

/** Presigns an upload of a new image of the user's review of the place. */
export async function requestReviewMediaUpload(
  placeId: number,
  contentType: string,
): Promise<MediaUpload> {
  return unwrap(
    await apiPost<MediaUpload>(
      `places/${placeId}/reviews/mine/medias/upload-url`,
      { json: { content_type: contentType } },
    ),
  );
}

/** Adds the uploaded image to the user's review of the place. */
export async function addReviewMedia(
  placeId: number,
  uploadKey: string,
): Promise<Review> {
  return unwrap(
    await apiPost<Review>(`places/${placeId}/reviews/mine/medias`, {
      json: { upload_key: uploadKey },
    }),
  );
}

export async function deleteReviewMedia(
  placeId: number,
  mediaId: number,
): Promise<void> {
  unwrap(
    await apiDelete<void>(`places/${placeId}/reviews/mine/medias/${mediaId}`),
  );
}

/**
 * Deletes the review and takes its rating off the place. `"mine"` is the
 * signed-in user's own review; any other review needs an admin.
 */
export async function deletePlaceReview(
  placeId: number,
  reviewId: number | "mine",
): Promise<void> {
  unwrap(await apiDelete<void>(`places/${placeId}/reviews/${reviewId}`));
}
