import type { SubDistrict } from "~/types/subdistricts";
import { api, apiResult, type ApiResult, type CommonApiErrorCode } from "./api";

// -----------------------------------------------------------------------------
// Error codes (service-layer concern, not a domain shape)
// -----------------------------------------------------------------------------

export type SubDistrictErrorCode = CommonApiErrorCode;

// -----------------------------------------------------------------------------
// Services (see internal/module/subdistrict/route.go)
// -----------------------------------------------------------------------------

export function getSubDistricts(): Promise<
  ApiResult<SubDistrict[], SubDistrictErrorCode>
> {
  return apiResult<SubDistrict[], SubDistrictErrorCode>(
    api.get("subdistricts/"),
  );
}
