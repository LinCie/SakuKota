import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { placeKeys } from "~/features/places/queries";

import {
  createPlaceReview,
  getPlaceReviews,
  type CreateReviewRequest,
} from "./api";

export const reviewKeys = {
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
