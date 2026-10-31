import {
  apiDelete,
  apiGet,
  apiPatch,
  apiPost,
  apiPut,
  type AdminWriteErrorCode,
  type CommonApiErrorCode,
  unwrap,
} from "~/shared/api";
import type { ImageErrorCode, ImageUpload } from "~/shared/image-upload";
import type { SubDistrict } from "./types";

// -----------------------------------------------------------------------------
// Request bodies / error codes (service-layer concerns, not domain shapes).
// Services resolve to the response data and throw an `ApiError` carrying one
// of these codes when the backend rejects the request.
// -----------------------------------------------------------------------------

export type SubDistrictErrorCode = CommonApiErrorCode;
export type CreateSubDistrictErrorCode =
  AdminWriteErrorCode | "VALIDATION_ERROR" | "SUB_DISTRICT_EXISTS";
export type GetSubDistrictErrorCode =
  CommonApiErrorCode | "INVALID_SUB_DISTRICT_ID" | "SUB_DISTRICT_NOT_FOUND";
export type UpdateSubDistrictErrorCode =
  | AdminWriteErrorCode
  | "INVALID_SUB_DISTRICT_ID"
  | "VALIDATION_ERROR"
  | "SUB_DISTRICT_NOT_FOUND"
  | "SUB_DISTRICT_EXISTS";
export type DeleteSubDistrictErrorCode =
  | AdminWriteErrorCode
  | "INVALID_SUB_DISTRICT_ID"
  | "SUB_DISTRICT_NOT_FOUND"
  | "SUB_DISTRICT_IN_USE";
export type SubDistrictImageErrorCode =
  ImageErrorCode | "INVALID_SUB_DISTRICT_ID" | "SUB_DISTRICT_NOT_FOUND";

/** One of a sub-district's images, named by its path segment. */
export type SubDistrictImageSlot = "image" | "static-map";

export type CreateSubDistrictRequest = {
  name: string;
  slug: string;
  flood_history?: string | null;
  safety_crime_information?: string | null;
  estimated_daily_spending?: number | null;
};

export type UpdateSubDistrictRequest = Partial<CreateSubDistrictRequest>;

// -----------------------------------------------------------------------------
// Services (see internal/router/router.go)
// -----------------------------------------------------------------------------

export async function getSubDistricts(): Promise<SubDistrict[]> {
  return unwrap(
    await apiGet<SubDistrict[], SubDistrictErrorCode>("subdistricts"),
  );
}

export async function getSubDistrict(id: number): Promise<SubDistrict> {
  return unwrap(
    await apiGet<SubDistrict, GetSubDistrictErrorCode>(`subdistricts/${id}`),
  );
}

/** Admin only. */
export async function createSubDistrict(
  body: CreateSubDistrictRequest,
): Promise<SubDistrict> {
  return unwrap(
    await apiPost<SubDistrict, CreateSubDistrictErrorCode>("subdistricts", {
      json: body,
    }),
  );
}

/** Admin only. */
export async function updateSubDistrict(
  id: number,
  body: UpdateSubDistrictRequest,
): Promise<SubDistrict> {
  return unwrap(
    await apiPatch<SubDistrict, UpdateSubDistrictErrorCode>(
      `subdistricts/${id}`,
      { json: body },
    ),
  );
}

/** Admin only. Fails with 409 while places reference the sub-district. */
export async function deleteSubDistrict(id: number): Promise<void> {
  unwrap(
    await apiDelete<void, DeleteSubDistrictErrorCode>(`subdistricts/${id}`),
  );
}

/** Admin only. Presigns an upload of a new image in the sub-district's `slot`. */
export async function requestSubDistrictImageUpload(
  id: number,
  slot: SubDistrictImageSlot,
  contentType: string,
): Promise<ImageUpload> {
  return unwrap(
    await apiPost<ImageUpload, SubDistrictImageErrorCode>(
      `subdistricts/${id}/${slot}/upload-url`,
      { json: { content_type: contentType } },
    ),
  );
}

/** Admin only. Makes the uploaded image the sub-district's image in `slot`. */
export async function setSubDistrictImage(
  id: number,
  slot: SubDistrictImageSlot,
  uploadKey: string,
): Promise<SubDistrict> {
  return unwrap(
    await apiPut<SubDistrict, SubDistrictImageErrorCode>(
      `subdistricts/${id}/${slot}`,
      { json: { upload_key: uploadKey } },
    ),
  );
}
