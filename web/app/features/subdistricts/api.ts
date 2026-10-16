import {
  apiGet,
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

export type CreateSubDistrictRequest = {
  name: string;
  slug: string;
  flood_history?: string | null;
  safety_crime_information?: string | null;
  estimated_daily_spending?: number | null;
  static_map_image_url?: string | null;
};

// -----------------------------------------------------------------------------
// Services (see internal/module/subdistrict/route.go)
// -----------------------------------------------------------------------------

export function getSubDistricts(): Promise<
  ApiResult<SubDistrict[], SubDistrictErrorCode>
> {
  return apiGet<SubDistrict[], SubDistrictErrorCode>("subdistricts");
}

/** Admin only. */
export function createSubDistrict(
  body: CreateSubDistrictRequest,
): Promise<ApiResult<SubDistrict, CreateSubDistrictErrorCode>> {
  return apiPost<SubDistrict, CreateSubDistrictErrorCode>("subdistricts", {
    json: body,
  });
}
