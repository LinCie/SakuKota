import { apiGet, unwrap } from "~/shared/api";
import type { SubDistrict } from "./types";

/** Every kecamatan (see internal/router/router.go). Throws an `ApiError` when the backend rejects the request. */
export async function getSubDistricts(): Promise<SubDistrict[]> {
  return unwrap(await apiGet<SubDistrict[]>("subdistricts"));
}
