import {
  apiDelete,
  apiGet,
  apiPatch,
  apiPost,
  type AdminWriteErrorCode,
  type ApiResult,
  type CommonApiErrorCode,
} from "~/shared/api";
import type { SubDistrict } from "./types";

// -----------------------------------------------------------------------------
// Request bodies / error codes (service-layer concerns, not domain shapes)
// -----------------------------------------------------------------------------

export type SubDistrictErrorCode = CommonApiErrorCode;
export type CreateSubDistrictErrorCode =
  AdminWriteErrorCode | "validation_error" | "sub_district_exists";
export type GetSubDistrictErrorCode =
  | CommonApiErrorCode
  | "invalid_sub_district_id"
  | "sub_district_not_found";
export type UpdateSubDistrictErrorCode =
  | AdminWriteErrorCode
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
// Services (see internal/module/subdistrict/route.go)
// -----------------------------------------------------------------------------

export function getSubDistricts(): Promise<
  ApiResult<SubDistrict[], SubDistrictErrorCode>
> {
  return apiGet<SubDistrict[], SubDistrictErrorCode>("subdistricts");
}

export function getSubDistrict(
  id: number,
): Promise<ApiResult<SubDistrict, GetSubDistrictErrorCode>> {
  return apiGet<SubDistrict, GetSubDistrictErrorCode>(`subdistricts/${id}`);
}

/** Admin only. */
export function createSubDistrict(
  body: CreateSubDistrictRequest,
): Promise<ApiResult<SubDistrict, CreateSubDistrictErrorCode>> {
  return apiPost<SubDistrict, CreateSubDistrictErrorCode>("subdistricts", {
    json: body,
  });
}

/** Admin only. */
export function updateSubDistrict(
  id: number,
  body: UpdateSubDistrictRequest,
): Promise<ApiResult<SubDistrict, UpdateSubDistrictErrorCode>> {
  return apiPatch<SubDistrict, UpdateSubDistrictErrorCode>(
    `subdistricts/${id}`,
    { json: body },
  );
}

/** Admin only. Fails with 409 while places reference the sub-district. */
export function deleteSubDistrict(
  id: number,
): Promise<ApiResult<void, DeleteSubDistrictErrorCode>> {
  return apiDelete<void, DeleteSubDistrictErrorCode>(`subdistricts/${id}`);
}
