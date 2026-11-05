import { apiGet, type CommonApiErrorCode, unwrap } from "~/shared/api";
import type { SubDistrict } from "./types";

// -----------------------------------------------------------------------------
// Error codes (service-layer concerns, not domain shapes).
// -----------------------------------------------------------------------------

export type SubDistrictErrorCode = CommonApiErrorCode;

// -----------------------------------------------------------------------------
// Services (see internal/router/router.go)
// -----------------------------------------------------------------------------

export async function getSubDistricts(): Promise<SubDistrict[]> {
  return unwrap(
    await apiGet<SubDistrict[], SubDistrictErrorCode>("subdistricts"),
  );
}
