import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { placeKeys } from "~/features/places/queries";
import { uploadImage } from "~/shared/image-upload";

import {
  addReviewMedia,
  createPlaceReview,
  deletePlaceReview,
  deleteReviewMedia,
  getPlaceReviews,
  requestReviewMediaUpload,
  type CreateReviewRequest,
} from "./api";

export const reviewKeys = {
  all: ["reviews"] as const,
  place: (placeId: number) => ["reviews", "place", placeId] as const,
};

// Queries. Reviews load when a place's details open, so they are not part of
// any route loader.

export function usePlaceReviews(placeId: number) {
  return useQuery({
    queryKey: reviewKeys.place(placeId),
    queryFn: () => getPlaceReviews(placeId),
  });
}

// Mutations. Resolves once the place's reviews and the places, whose ratings
// changed, refetched.

export function useCreateReview(placeId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateReviewRequest) => createPlaceReview(placeId, body),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: reviewKeys.place(placeId) }),
        queryClient.invalidateQueries({ queryKey: placeKeys.all }),
      ]),
  });
}

export function useDeleteReview(placeId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reviewId: number | "mine") =>
      deletePlaceReview(placeId, reviewId),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: reviewKeys.place(placeId) }),
        queryClient.invalidateQueries({ queryKey: placeKeys.all }),
      ]),
  });
}

/**
 * Uploads `files` one by one and adds each to the user's review of the place.
 * Refetches the reviews even after a failure, so the files added so far show.
 */
export function useAddReviewMedias(placeId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (files: File[]) => {
      for (const file of files) {
        const uploadKey = await uploadImage(file, (contentType) =>
          requestReviewMediaUpload(placeId, contentType),
        );
        await addReviewMedia(placeId, uploadKey);
      }
    },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: reviewKeys.place(placeId) }),
  });
}

export function useDeleteReviewMedia(placeId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mediaId: number) => deleteReviewMedia(placeId, mediaId),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: reviewKeys.place(placeId) }),
  });
}
