import {
  queryOptions,
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";

import {
  createSubDistrict,
  deleteSubDistrict,
  getSubDistrict,
  getSubDistricts,
  requestSubDistrictImageUpload,
  setSubDistrictImage,
  updateSubDistrict,
  type SubDistrictImageSlot,
  type UpdateSubDistrictRequest,
} from "./api";
import { uploadImage } from "~/shared/image-upload";

import type { SubDistrict } from "./types";

// -----------------------------------------------------------------------------
// Query keys
// -----------------------------------------------------------------------------

export const subDistrictKeys = {
  all: ["subdistricts"] as const,
  lists: () => [...subDistrictKeys.all, "list"] as const,
  detail: (id: number) => [...subDistrictKeys.all, "detail", id] as const,
};

// -----------------------------------------------------------------------------
// Queries. The options are shared by route loaders (`query`) and the
// hooks below, which read the data the loader put in the cache.
// -----------------------------------------------------------------------------

export function subDistrictsQuery() {
  return queryOptions({
    queryKey: subDistrictKeys.lists(),
    queryFn: getSubDistricts,
  });
}

export function subDistrictQuery(id: number) {
  return queryOptions({
    queryKey: subDistrictKeys.detail(id),
    queryFn: () => getSubDistrict(id),
  });
}

export function useSubDistricts() {
  return useSuspenseQuery(subDistrictsQuery());
}

export function useSubDistrict(id: number) {
  return useSuspenseQuery(subDistrictQuery(id));
}

// -----------------------------------------------------------------------------
// Mutations (admin only). Each resolves once the affected queries refetched.
// -----------------------------------------------------------------------------

export function useCreateSubDistrict() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createSubDistrict,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: subDistrictKeys.all }),
  });
}

export function useUpdateSubDistrict() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      body,
    }: {
      id: number;
      body: UpdateSubDistrictRequest;
    }) => updateSubDistrict(id, body),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: subDistrictKeys.all }),
  });
}

/** Uploads `file` and makes it the sub-district's image in `slot`. */
export function useUploadSubDistrictImage(slot: SubDistrictImageSlot) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, file }: { id: number; file: File }) => {
      const uploadKey = await uploadImage(file, (contentType) =>
        requestSubDistrictImageUpload(id, slot, contentType),
      );
      return setSubDistrictImage(id, slot, uploadKey);
    },
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: subDistrictKeys.all }),
  });
}

export function useDeleteSubDistrict() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deleteSubDistrict,
    onSuccess: (_data, id) => {
      // Drop it from cached lists at once; a detail page on screen removes
      // its own query after navigating away.
      queryClient.setQueriesData<SubDistrict[]>(
        { queryKey: subDistrictKeys.lists() },
        (subDistricts) => subDistricts?.filter((sd) => sd.id !== id),
      );
      queryClient.removeQueries({
        queryKey: subDistrictKeys.detail(id),
        type: "inactive",
      });
      return queryClient.invalidateQueries({
        queryKey: subDistrictKeys.lists(),
      });
    },
  });
}
