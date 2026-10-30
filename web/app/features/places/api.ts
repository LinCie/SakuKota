import {
  apiDelete,
  apiGet,
  apiPatch,
  apiPost,
  type AdminWriteErrorCode,
  type CommonApiErrorCode,
  unwrap,
} from "~/shared/api";
import type { CategoryDetail, GeoPoint, Place, PlacePage } from "./types";

// -----------------------------------------------------------------------------
// Request params / error codes (service-layer concerns, not domain shapes).
// Services resolve to the response data and throw an `ApiError` carrying one
// of these codes when the backend rejects the request.
// -----------------------------------------------------------------------------

export type PlaceErrorCode =
  | CommonApiErrorCode
  | "invalid_sub_district_id"
  | "invalid_category_id"
  | "invalid_limit"
  | "invalid_bbox"
  | "sub_district_not_found"
  | "category_not_found";
export type GetPlaceErrorCode =
  CommonApiErrorCode | "invalid_place_id" | "place_not_found";
export type CategoryErrorCode = CommonApiErrorCode;
export type GetCategoryErrorCode =
  CommonApiErrorCode | "invalid_category_id" | "category_not_found";
export type UpdatePlaceErrorCode =
  | AdminWriteErrorCode
  | "invalid_place_id"
  | "validation_error"
  | "place_not_found";
export type DeletePlaceErrorCode =
  AdminWriteErrorCode | "invalid_place_id" | "place_not_found";
export type CreatePlaceErrorCode = AdminWriteErrorCode | "validation_error";

export type GetPlacesParams = {
  /** Filters places to one sub-district, matching the `sub_district_id` query param. */
  subDistrictId?: number;
  /** Keeps places in any of these categories, matching the `category_id` query param. */
  categoryIds?: number[];
  /** Case-insensitive search term, matching the `q` query param. */
  q?: string;
  /** Keeps only the lowest-id places, matching the `limit` query param. */
  limit?: number;
  /** Keeps places inside "west,south,east,north" (degrees), matching the `bbox` query param. */
  bbox?: string;
};

export type CreatePlaceRequest = {
  sub_district_id: number;
  category_id: number;
  name: string;
  address?: string | null;
  location?: GeoPoint | null;
  operating_hours?: string | null;
  estimated_price_range?: string | null;
  short_description?: string | null;
  /** Values of the category's custom fields, keyed by `CategoryField.key`. */
  attributes?: Record<string, unknown>;
};

export type UpdatePlaceRequest = {
  sub_district_id?: number;
  category_id?: number;
  name?: string;
  address?: string | null;
  location?: GeoPoint | null;
  operating_hours?: string | null;
  estimated_price_range?: string | null;
  short_description?: string | null;
  /** Replaces all attributes when set; must match the category's fields. */
  attributes?: Record<string, unknown> | null;
};

// -----------------------------------------------------------------------------
// Services (see internal/router/router.go)
// -----------------------------------------------------------------------------

export async function getPlaces(
  params: GetPlacesParams = {},
): Promise<PlacePage> {
  const { subDistrictId, categoryIds, q, limit, bbox } = params;
  const searchParams = new URLSearchParams();
  if (subDistrictId !== undefined) {
    searchParams.set("sub_district_id", String(subDistrictId));
  }
  if (categoryIds?.length)
    searchParams.set("category_id", categoryIds.join(","));
  if (q) searchParams.set("q", q);
  if (limit !== undefined) searchParams.set("limit", String(limit));
  if (bbox) searchParams.set("bbox", bbox);

  return unwrap(
    await apiGet<PlacePage, PlaceErrorCode>("places", { searchParams }),
  );
}

export async function getPlace(id: number): Promise<Place> {
  return unwrap(await apiGet<Place, GetPlaceErrorCode>(`places/${id}`));
}

export async function getCategories(): Promise<CategoryDetail[]> {
  return unwrap(
    await apiGet<CategoryDetail[], CategoryErrorCode>("places/categories"),
  );
}

export async function getCategory(id: number): Promise<CategoryDetail> {
  return unwrap(
    await apiGet<CategoryDetail, GetCategoryErrorCode>(
      `places/categories/${id}`,
    ),
  );
}

/** Admin only. */
export async function createPlace(body: CreatePlaceRequest): Promise<Place> {
  return unwrap(
    await apiPost<Place, CreatePlaceErrorCode>("places", { json: body }),
  );
}

/** Admin only. */
export async function updatePlace(
  id: number,
  body: UpdatePlaceRequest,
): Promise<Place> {
  return unwrap(
    await apiPatch<Place, UpdatePlaceErrorCode>(`places/${id}`, {
      json: body,
    }),
  );
}

/** Admin only. */
export async function deletePlace(id: number): Promise<void> {
  unwrap(await apiDelete<void, DeletePlaceErrorCode>(`places/${id}`));
}
