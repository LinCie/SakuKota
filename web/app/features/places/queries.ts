import {
  queryOptions,
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";

import {
  createPlace,
  deletePlace,
  getCategories,
  getPlace,
  getPlaces,
  requestPlaceImageUpload,
  setPlaceImage,
  updatePlace,
  type GetPlacesParams,
  type UpdatePlaceRequest,
} from "./api";
import { uploadImage } from "~/shared/image-upload";

import type { PlacePage } from "./types";

// -----------------------------------------------------------------------------
// Query keys
// -----------------------------------------------------------------------------

export const placeKeys = {
  all: ["places"] as const,
  lists: () => [...placeKeys.all, "list"] as const,
  list: (params: GetPlacesParams) => [...placeKeys.lists(), params] as const,
  detail: (id: number) => [...placeKeys.all, "detail", id] as const,
};

export const categoryKeys = {
  all: ["categories"] as const,
  lists: () => [...categoryKeys.all, "list"] as const,
};

// -----------------------------------------------------------------------------
// Queries. The options are shared by route loaders (`query`) and the
// hooks below, which read the data the loader put in the cache.
// -----------------------------------------------------------------------------

export function placesQuery(params: GetPlacesParams = {}) {
  return queryOptions({
    queryKey: placeKeys.list(params),
    queryFn: () => getPlaces(params),
  });
}

export function placeQuery(id: number) {
  return queryOptions({
    queryKey: placeKeys.detail(id),
    queryFn: () => getPlace(id),
  });
}

export function categoriesQuery() {
  return queryOptions({
    queryKey: categoryKeys.lists(),
    queryFn: getCategories,
  });
}

export function usePlaces(params: GetPlacesParams = {}) {
  return useSuspenseQuery(placesQuery(params));
}

export function usePlace(id: number) {
  return useSuspenseQuery(placeQuery(id));
}

export function useCategories() {
  return useSuspenseQuery(categoriesQuery());
}

// -----------------------------------------------------------------------------
// Mutations (admin only). Each resolves once the affected queries refetched.
// -----------------------------------------------------------------------------

export function useCreatePlace() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createPlace,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: placeKeys.all }),
  });
}

/** Uploads `file` and makes it the place's image. */
export function useUploadPlaceImage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, file }: { id: number; file: File }) => {
      const uploadKey = await uploadImage(file, (contentType) =>
        requestPlaceImageUpload(id, contentType),
      );
      return setPlaceImage(id, uploadKey);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: placeKeys.all }),
  });
}

export function useUpdatePlace() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: UpdatePlaceRequest }) =>
      updatePlace(id, body),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: placeKeys.all }),
  });
}

export function useDeletePlace() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deletePlace,
    onSuccess: (_data, id) => {
      // Drop it from every cached list at once, so lists not on screen don't
      // show it until they refetch. A detail page on screen removes its own
      // query after navigating away.
      queryClient.setQueriesData<PlacePage>(
        { queryKey: placeKeys.lists() },
        (page) =>
          page?.places.some((place) => place.id === id)
            ? {
                places: page.places.filter((place) => place.id !== id),
                total: page.total - 1,
              }
            : page,
      );
      queryClient.removeQueries({
        queryKey: placeKeys.detail(id),
        type: "inactive",
      });
      return queryClient.invalidateQueries({ queryKey: placeKeys.lists() });
    },
  });
}
