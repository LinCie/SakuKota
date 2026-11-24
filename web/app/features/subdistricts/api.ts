import { apiDelete, apiGet, apiPost, apiPut, unwrap } from "~/shared/api";
import type { MediaUpload } from "~/shared/image-upload";
import type { Boundary, SubDistrict } from "./types";

// Every function here calls one backend route (see internal/router/router.go),
// returns the response data, and throws an `ApiError` when the backend rejects
// the request.

/** A kecamatan's details. Replacing one clears the fields left out. */
export type SubDistrictRequest = {
  name: string;
  slug: string;
  flood_history: string | null;
  safety_crime_information: string | null;
  estimated_daily_spending: number | null;
};

/** Every kecamatan. */
export async function getSubDistricts(): Promise<SubDistrict[]> {
  return unwrap(await apiGet<SubDistrict[]>("subdistricts"));
}

export async function getSubDistrict(id: number): Promise<SubDistrict> {
  return unwrap(await apiGet<SubDistrict>(`subdistricts/${id}`));
}

/** Admin only. */
export async function createSubDistrict(
  body: SubDistrictRequest,
): Promise<SubDistrict> {
  return unwrap(await apiPost<SubDistrict>("subdistricts", { json: body }));
}

/** Admin only. Keeps the kecamatan's images and boundary. */
export async function updateSubDistrict(
  id: number,
  body: SubDistrictRequest,
): Promise<SubDistrict> {
  return unwrap(
    await apiPut<SubDistrict>(`subdistricts/${id}`, { json: body }),
  );
}

/** Admin only. Fails with `SUB_DISTRICT_IN_USE` while places are in it. */
export async function deleteSubDistrict(id: number): Promise<void> {
  unwrap(await apiDelete<void>(`subdistricts/${id}`));
}

/** Admin only. Replaces the kecamatan's boundary. */
export async function setSubDistrictBoundary(
  id: number,
  boundary: Boundary,
): Promise<SubDistrict> {
  return unwrap(
    await apiPut<SubDistrict>(`subdistricts/${id}/boundary`, {
      json: boundary,
    }),
  );
}

/** Admin only. Presigns an upload of a new image of the kecamatan. */
export async function requestSubDistrictMediaUpload(
  id: number,
  contentType: string,
): Promise<MediaUpload> {
  return unwrap(
    await apiPost<MediaUpload>(`subdistricts/${id}/medias/upload-url`, {
      json: { content_type: contentType },
    }),
  );
}

/** Admin only. Adds the uploaded image to the kecamatan's medias. */
export async function addSubDistrictMedia(
  id: number,
  uploadKey: string,
): Promise<SubDistrict> {
  return unwrap(
    await apiPost<SubDistrict>(`subdistricts/${id}/medias`, {
      json: { upload_key: uploadKey },
    }),
  );
}

/** Admin only. */
export async function deleteSubDistrictMedia(
  id: number,
  mediaId: number,
): Promise<void> {
  unwrap(await apiDelete<void>(`subdistricts/${id}/medias/${mediaId}`));
}
