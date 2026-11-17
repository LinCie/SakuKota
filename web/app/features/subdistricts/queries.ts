import { queryOptions, useSuspenseQuery } from "@tanstack/react-query";

import { getSubDistricts } from "./api";

export const subDistrictKeys = {
  lists: () => ["subdistricts", "list"] as const,
};

// Queries. The options are shared by route loaders (`query`) and the
// hooks below, which read the data the loader put in the cache.

export function subDistrictsQuery() {
  return queryOptions({
    queryKey: subDistrictKeys.lists(),
    queryFn: getSubDistricts,
  });
}

export function useSubDistricts() {
  return useSuspenseQuery(subDistrictsQuery());
}
