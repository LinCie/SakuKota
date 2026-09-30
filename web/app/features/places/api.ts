import {
  api,
  apiResult,
  type ApiResult,
  type CommonApiErrorCode,
} from "~/shared/api";
import type { CategoryDetail, Place } from "./types";

// -----------------------------------------------------------------------------
// Request params / error codes (service-layer concerns, not domain shapes)
// -----------------------------------------------------------------------------

export type PlaceErrorCode =
  CommonApiErrorCode | "invalid_sub_district_id" | "invalid_category_id";
export type CategoryErrorCode = CommonApiErrorCode;

export type GetPlacesParams = {
  /** Filters places to one sub-district, matching the `sub_district_id` query param. */
  subDistrictId?: number;
  /** Keeps places in any of these categories, matching the `category_id` query param. */
  categoryIds?: number[];
  /** Case-insensitive search term, matching the `q` query param. */
  q?: string;
};

// -----------------------------------------------------------------------------
// Services (see internal/module/place/route.go)
// -----------------------------------------------------------------------------

export function getPlaces(
  params: GetPlacesParams = {},
): Promise<ApiResult<Place[], PlaceErrorCode>> {
  const { subDistrictId, categoryIds, q } = params;
  const searchParams = new URLSearchParams();
  if (subDistrictId !== undefined) {
    searchParams.set("sub_district_id", String(subDistrictId));
  }
  if (categoryIds?.length)
    searchParams.set("category_id", categoryIds.join(","));
  if (q) searchParams.set("q", q);

  return apiResult<Place[], PlaceErrorCode>(
    api.get("places/", { searchParams }),
  );
}

export function getCategories(): Promise<
  ApiResult<CategoryDetail[], CategoryErrorCode>
> {
  return apiResult<CategoryDetail[], CategoryErrorCode>(
    api.get("places/categories/"),
  );
}
