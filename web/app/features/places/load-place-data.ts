import type { QueryClient } from "@tanstack/react-query";

import {
  subDistrictsQuery,
  useSubDistricts,
} from "~/features/subdistricts/queries";

import {
  categoriesQuery,
  placesQuery,
  useCategories,
  usePlaces,
} from "./queries";

/** Loads everything the places pages need, for route loaders. */
export async function ensurePlaceData(queryClient: QueryClient) {
  await Promise.all([
    queryClient.ensureQueryData(categoriesQuery()),
    queryClient.ensureQueryData(subDistrictsQuery()),
    queryClient.ensureQueryData(placesQuery()),
  ]);
}

/** Everything the places pages need: the places, their kecamatan, and categories with fields. */
export function usePlaceData() {
  const { data: categories } = useCategories();
  const { data: subDistricts } = useSubDistricts();
  const { data: places } = usePlaces();
  return { categories, subDistricts, places };
}
