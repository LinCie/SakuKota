import {
  infiniteQueryOptions,
  queryOptions,
  type QueryClient,
  useMutation,
  useQueryClient,
  useSuspenseInfiniteQuery,
  useSuspenseQuery,
} from "@tanstack/react-query";

import {
  addPlaceMedia,
  createPlace,
  deletePlace,
  deletePlaceMedia,
  getCategories,
  getPlace,
  getPlaces,
  requestPlaceMediaUpload,
  updatePlace,
  type GetPlacesParams,
  type PlaceRequest,
} from "./api";
import {
  subDistrictsQuery,
  useSubDistricts,
} from "~/features/subdistricts/queries";
import { uploadImage } from "~/shared/image-upload";

import type { PlacePage } from "./types";

/** The filters of a list that loads page by page; the page is not part of them. */
export type PlaceListParams = Omit<
  GetPlacesParams,
  "limit" | "offset" | "bbox"
>;

const PLACES_PAGE_SIZE = 20;

// Query keys: the names the fetched data is cached under. Invalidating a key
// refetches every key that starts with it, so `all` covers everything below.

export const placeKeys = {
  all: ["places"] as const,
  lists: () => ["places", "list"] as const,
  list: (params: GetPlacesParams) => ["places", "list", params] as const,
  // Not under "list": paged data has another shape than a `PlacePage`.
  infinite: (params: PlaceListParams) =>
    ["places", "infinite", params] as const,
  detail: (id: number) => ["places", "detail", id] as const,
};

export const categoryKeys = {
  lists: () => ["categories", "list"] as const,
};

// Queries. The options are shared by route loaders (`query`) and the
// hooks below, which read the data the loader put in the cache.

export function placesQuery(params: GetPlacesParams = {}) {
  return queryOptions({
    queryKey: placeKeys.list(params),
    queryFn: () => getPlaces(params),
  });
}

/** The places matching the filters, loaded page by page. */
export function placesInfiniteQuery(params: PlaceListParams = {}) {
  return infiniteQueryOptions({
    queryKey: placeKeys.infinite(params),
    queryFn: ({ pageParam }) =>
      getPlaces({ ...params, limit: PLACES_PAGE_SIZE, offset: pageParam }),
    initialPageParam: 0,
    // The next page starts after the places loaded so far; `undefined` means
    // there is nothing more to load.
    getNextPageParam: (lastPage, pages) => {
      let loaded = 0;
      for (const page of pages) loaded += page.places.length;

      const hasMore = lastPage.places.length > 0 && loaded < lastPage.total;
      return hasMore ? loaded : undefined;
    },
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

export function usePlacesInfinite(params: PlaceListParams = {}) {
  return useSuspenseInfiniteQuery(placesInfiniteQuery(params));
}

export function usePlace(id: number) {
  return useSuspenseQuery(placeQuery(id));
}

export function useCategories() {
  return useSuspenseQuery(categoriesQuery());
}

/** Loads everything the places pages need, for route loaders. */
export async function ensurePlaceData(queryClient: QueryClient) {
  await Promise.all([
    queryClient.query({ ...categoriesQuery(), staleTime: "static" }),
    queryClient.query({ ...subDistrictsQuery(), staleTime: "static" }),
    queryClient.query({ ...placesQuery(), staleTime: "static" }),
  ]);
}

/** Everything the places pages need: the places, their kecamatan, and categories with fields. */
export function usePlaceData() {
  const { data: categories } = useCategories();
  const { data: subDistricts } = useSubDistricts();
  const {
    data: { places },
  } = usePlaces();
  return { categories, subDistricts, places };
}

// Mutations (admin only). Each resolves once the affected queries refetched.

export function useCreatePlace() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createPlace,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: placeKeys.all }),
  });
}

/** Uploads `file` and adds it to the place's medias. */
export function useAddPlaceMedia() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, file }: { id: number; file: File }) => {
      const uploadKey = await uploadImage(file, (contentType) =>
        requestPlaceMediaUpload(id, contentType),
      );
      return addPlaceMedia(id, uploadKey);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: placeKeys.all }),
  });
}

export function useDeletePlaceMedia() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, mediaId }: { id: number; mediaId: number }) =>
      deletePlaceMedia(id, mediaId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: placeKeys.all }),
  });
}

export function useUpdatePlace() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: PlaceRequest }) =>
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
        (page) => {
          if (!page) return page;
          const hasPlace = page.places.some((place) => place.id === id);
          if (!hasPlace) return page;
          return {
            places: page.places.filter((place) => place.id !== id),
            total: page.total - 1,
          };
        },
      );
      queryClient.removeQueries({
        queryKey: placeKeys.detail(id),
        type: "inactive",
      });
      return queryClient.invalidateQueries({ queryKey: placeKeys.lists() });
    },
  });
}
