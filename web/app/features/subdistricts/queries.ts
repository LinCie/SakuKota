import {
  queryOptions,
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";

import { uploadImage } from "~/shared/image-upload";

import {
  addSubDistrictMedia,
  createSubDistrict,
  deleteSubDistrict,
  deleteSubDistrictMedia,
  getSubDistrict,
  getSubDistricts,
  requestSubDistrictMediaUpload,
  setSubDistrictBoundary,
  updateSubDistrict,
  type SubDistrictRequest,
} from "./api";
import type { Boundary } from "./types";

// Query keys: invalidating `all` refetches every key that starts with it.

export const subDistrictKeys = {
  all: ["subdistricts"] as const,
  lists: () => ["subdistricts", "list"] as const,
  detail: (id: number) => ["subdistricts", "detail", id] as const,
};

// Queries. The options are shared by route loaders (`query`) and the
// hooks below, which read the data the loader put in the cache.

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

// Mutations (admin only). Each resolves once the affected queries refetched.

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
    mutationFn: ({ id, body }: { id: number; body: SubDistrictRequest }) =>
      updateSubDistrict(id, body),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: subDistrictKeys.all }),
  });
}

export function useSetSubDistrictBoundary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, boundary }: { id: number; boundary: Boundary }) =>
      setSubDistrictBoundary(id, boundary),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: subDistrictKeys.all }),
  });
}

/** Uploads `file` and adds it to the kecamatan's medias. */
export function useAddSubDistrictMedia() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, file }: { id: number; file: File }) => {
      const uploadKey = await uploadImage(file, (contentType) =>
        requestSubDistrictMediaUpload(id, contentType),
      );
      return addSubDistrictMedia(id, uploadKey);
    },
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: subDistrictKeys.all }),
  });
}

export function useDeleteSubDistrictMedia() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, mediaId }: { id: number; mediaId: number }) =>
      deleteSubDistrictMedia(id, mediaId),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: subDistrictKeys.all }),
  });
}

export function useDeleteSubDistrict() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deleteSubDistrict,
    onSuccess: (_data, id) => {
      // A detail page on screen removes its own query after navigating away.
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
