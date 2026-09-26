import type { Category, Place } from "~/types/places";
import { api, apiResult, type ApiResult, type CommonApiErrorCode } from "./api";

// -----------------------------------------------------------------------------
// Request params / error codes (service-layer concerns, not domain shapes)
// -----------------------------------------------------------------------------

export type PlaceErrorCode = CommonApiErrorCode | "invalid_category_id";
export type CategoryErrorCode = CommonApiErrorCode;

export type GetPlacesParams = {
  /** Filters places down to a single category, matching the `category_id` query param. */
  categoryId?: number;
};

// -----------------------------------------------------------------------------
// Services (see internal/module/place/route.go)
// -----------------------------------------------------------------------------

export function getPlaces(
  params: GetPlacesParams = {},
): Promise<ApiResult<Place[], PlaceErrorCode>> {
  const { categoryId } = params;

  return apiResult<Place[], PlaceErrorCode>(
    api.get("places/", {
      searchParams:
        categoryId !== undefined ? { category_id: categoryId } : undefined,
    }),
  );
}

export function getCategories(): Promise<
  ApiResult<Category[], CategoryErrorCode>
> {
  return apiResult<Category[], CategoryErrorCode>(
    api.get("places/categories/"),
  );
}
