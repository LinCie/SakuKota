import {
  apiDelete,
  apiGet,
  apiPatch,
  apiPost,
  type AdminWriteErrorCode,
  type CommonApiErrorCode,
  unwrap,
} from "~/shared/api";
import type { SubDistrict } from "./types";

// -----------------------------------------------------------------------------
// Request bodies / error codes (service-layer concerns, not domain shapes).
// Services resolve to the response data and throw an `ApiError` carrying one
// of these codes when the backend rejects the request.
// -----------------------------------------------------------------------------

export type SubDistrictErrorCode = CommonApiErrorCode;
export type CreateSubDistrictErrorCode =
  AdminWriteErrorCode | "validation_error" | "sub_district_exists";
export type GetSubDistrictErrorCode =
  CommonApiErrorCode | "invalid_sub_district_id" | "sub_district_not_found";
export type UpdateSubDistrictErrorCode =
  | AdminWriteErrorCode
  | "invalid_sub_district_id"
  | "validation_error"
  | "sub_district_not_found"
  | "sub_district_exists";
export type DeleteSubDistrictErrorCode =
  | AdminWriteErrorCode
  | "invalid_sub_district_id"
  | "sub_district_not_found"
  | "sub_district_in_use";

export type CreateSubDistrictRequest = {
  name: string;
  slug: string;
  flood_history?: string | null;
  safety_crime_information?: string | null;
  estimated_daily_spending?: number | null;
  static_map_image_url?: string | null;
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
