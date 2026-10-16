import {
  apiGet,
  apiPost,
  type AdminWriteErrorCode,
  type ApiResult,
  type CommonApiErrorCode,
} from "~/shared/api";
import type {
  CategoryDetail,
  CategoryField,
  FieldDataType,
  GeoPoint,
  Place,
} from "./types";

// -----------------------------------------------------------------------------
// Request params / error codes (service-layer concerns, not domain shapes)
// -----------------------------------------------------------------------------

export type PlaceErrorCode =
  CommonApiErrorCode | "invalid_sub_district_id" | "invalid_category_id";
export type CategoryErrorCode = CommonApiErrorCode;
export type CreatePlaceErrorCode = AdminWriteErrorCode | "validation_error";
export type CreateCategoryErrorCode =
  AdminWriteErrorCode | "validation_error" | "category_exists";
export type CreateCategoryFieldErrorCode =
  | AdminWriteErrorCode
  | "invalid_category_id"
  | "validation_error"
  | "category_not_found"
  | "category_field_exists";

export type GetPlacesParams = {
  /** Filters places to one sub-district, matching the `sub_district_id` query param. */
  subDistrictId?: number;
  /** Keeps places in any of these categories, matching the `category_id` query param. */
  categoryIds?: number[];
  /** Case-insensitive search term, matching the `q` query param. */
  q?: string;
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

export type CreateCategoryRequest = {
  name: string;
  slug: string;
};

export type CreateCategoryFieldRequest = {
  key: string;
  label: string;
  data_type: FieldDataType;
  required?: boolean;
  /** Allowed values; required for select fields, rejected for others. */
  options?: string[];
  /** 1-based position to insert the field at; omitted, the field goes last. */
  sort_order?: number;
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

  return apiGet<Place[], PlaceErrorCode>("places", { searchParams });
}

export function getCategories(): Promise<
  ApiResult<CategoryDetail[], CategoryErrorCode>
> {
  return apiGet<CategoryDetail[], CategoryErrorCode>("places/categories");
}

/** Admin only. */
export function createPlace(
  body: CreatePlaceRequest,
): Promise<ApiResult<Place, CreatePlaceErrorCode>> {
  return apiPost<Place, CreatePlaceErrorCode>("places", { json: body });
}

/** Admin only. */
export function createCategory(
  body: CreateCategoryRequest,
): Promise<ApiResult<CategoryDetail, CreateCategoryErrorCode>> {
  return apiPost<CategoryDetail, CreateCategoryErrorCode>("places/categories", {
    json: body,
  });
}

/** Admin only. */
export function createCategoryField(
  categoryId: number,
  body: CreateCategoryFieldRequest,
): Promise<ApiResult<CategoryField, CreateCategoryFieldErrorCode>> {
  return apiPost<CategoryField, CreateCategoryFieldErrorCode>(
    `places/categories/${categoryId}/fields`,
    { json: body },
  );
}
