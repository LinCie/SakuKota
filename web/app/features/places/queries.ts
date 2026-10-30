import {
  queryOptions,
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";

import {
  createCategory,
  createCategoryField,
  createPlace,
  deleteCategory,
  deletePlace,
  getCategories,
  getCategory,
  getPlace,
  getPlaces,
  updateCategory,
  updatePlace,
  type CreateCategoryFieldRequest,
  type GetPlacesParams,
  type UpdateCategoryRequest,
  type UpdatePlaceRequest,
} from "./api";
import type { CategoryDetail, Place } from "./types";

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
  detail: (id: number) => [...categoryKeys.all, "detail", id] as const,
};

// -----------------------------------------------------------------------------
// Queries. The options are shared by route loaders (`ensureQueryData`) and the
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

export function categoryQuery(id: number) {
  return queryOptions({
    queryKey: categoryKeys.detail(id),
    queryFn: () => getCategory(id),
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

export function useCategory(id: number) {
  return useSuspenseQuery(categoryQuery(id));
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
      queryClient.setQueriesData<Place[]>(
        { queryKey: placeKeys.lists() },
        (places) => places?.filter((place) => place.id !== id),
      );
      queryClient.removeQueries({
        queryKey: placeKeys.detail(id),
        type: "inactive",
      });
      return queryClient.invalidateQueries({ queryKey: placeKeys.lists() });
    },
  });
}

export function useCreateCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createCategory,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: categoryKeys.all }),
  });
}

export function useUpdateCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: UpdateCategoryRequest }) =>
      updateCategory(id, body),
    // Places embed their category's name and slug.
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: categoryKeys.all }),
        queryClient.invalidateQueries({ queryKey: placeKeys.all }),
      ]),
  });
}

export function useDeleteCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deleteCategory,
    onSuccess: (_data, id) => {
      queryClient.setQueriesData<CategoryDetail[]>(
        { queryKey: categoryKeys.lists() },
        (categories) => categories?.filter((category) => category.id !== id),
      );
      queryClient.removeQueries({
        queryKey: categoryKeys.detail(id),
        type: "inactive",
      });
      return queryClient.invalidateQueries({ queryKey: categoryKeys.lists() });
    },
  });
}

export function useCreateCategoryField() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      categoryId,
      body,
    }: {
      categoryId: number;
      body: CreateCategoryFieldRequest;
    }) => createCategoryField(categoryId, body),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: categoryKeys.all }),
  });
}
