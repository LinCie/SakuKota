import {
  apiDelete,
  apiGet,
  apiPost,
  apiPut,
  type AdminWriteErrorCode,
  type CommonApiErrorCode,
  unwrap,
} from "~/shared/api";
import type { ImageErrorCode, ImageUpload } from "~/shared/image-upload";
import type { CategoryDetail, GeoPoint, Place, PlacePage } from "./types";

// -----------------------------------------------------------------------------
// Request params / error codes (service-layer concerns, not domain shapes).
// Services resolve to the response data and throw an `ApiError` carrying one
// of these codes when the backend rejects the request.
// -----------------------------------------------------------------------------

export type PlaceErrorCode =
  | CommonApiErrorCode
  | "INVALID_SUB_DISTRICT_ID"
  | "INVALID_CATEGORY_ID"
  | "INVALID_LIMIT"
  | "INVALID_OFFSET"
  | "INVALID_BBOX"
  | "SUB_DISTRICT_NOT_FOUND"
  | "CATEGORY_NOT_FOUND";
export type GetPlaceErrorCode =
  CommonApiErrorCode | "INVALID_PLACE_ID" | "PLACE_NOT_FOUND";
export type CategoryErrorCode = CommonApiErrorCode;
export type UpdatePlaceErrorCode =
  | AdminWriteErrorCode
  | "INVALID_PLACE_ID"
  | "VALIDATION_ERROR"
  | "PLACE_NOT_FOUND";
export type DeletePlaceErrorCode =
  AdminWriteErrorCode | "INVALID_PLACE_ID" | "PLACE_NOT_FOUND";
export type CreatePlaceErrorCode = AdminWriteErrorCode | "VALIDATION_ERROR";
export type PlaceImageErrorCode =
  ImageErrorCode | "INVALID_PLACE_ID" | "PLACE_NOT_FOUND";

export type GetPlacesParams = {
  /** Filters places to one sub-district, matching the `sub_district_id` query param. */
  subDistrictId?: number;
  /** Keeps places in any of these categories, matching the `category_id` query param. */
  categoryIds?: number[];
  /** Case-insensitive search term, matching the `q` query param. */
  q?: string;
  /** Keeps only the lowest-id places, matching the `limit` query param. */
  limit?: number;
  /** Skips the lowest-id places before `limit` applies, matching the `offset` query param. */
  offset?: number;
  /** Keeps places inside "west,south,east,north" (degrees), matching the `bbox` query param. */
  bbox?: string;
};

export type PlaceRequest = {
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

// -----------------------------------------------------------------------------
// Services (see internal/router/router.go)
// -----------------------------------------------------------------------------

export async function getPlaces(
  params: GetPlacesParams = {},
): Promise<PlacePage> {
  const { subDistrictId, categoryIds, q, limit, offset, bbox } = params;
  const searchParams = new URLSearchParams();
  if (subDistrictId !== undefined) {
    searchParams.set("sub_district_id", String(subDistrictId));
  }
  if (categoryIds?.length)
    searchParams.set("category_id", categoryIds.join(","));
  if (q) searchParams.set("q", q);
  if (limit !== undefined) searchParams.set("limit", String(limit));
  if (offset) searchParams.set("offset", String(offset));
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

/** Admin only. */
export async function createPlace(body: PlaceRequest): Promise<Place> {
  return unwrap(
    await apiPost<Place, CreatePlaceErrorCode>("places", { json: body }),
  );
}

/** Admin only. */
export async function updatePlace(
  id: number,
  body: PlaceRequest,
): Promise<Place> {
  return unwrap(
    await apiPut<Place, UpdatePlaceErrorCode>(`places/${id}`, {
      json: body,
    }),
  );
}

/** Admin only. */
export async function deletePlace(id: number): Promise<void> {
  unwrap(await apiDelete<void, DeletePlaceErrorCode>(`places/${id}`));
}

/** Admin only. Presigns an upload of a new image of the place. */
export async function requestPlaceImageUpload(
  id: number,
  contentType: string,
): Promise<ImageUpload> {
  return unwrap(
    await apiPost<ImageUpload, PlaceImageErrorCode>(
      `places/${id}/image/upload-url`,
      { json: { content_type: contentType } },
    ),
  );
}

/** Admin only. Makes the uploaded image the place's image. */
export async function setPlaceImage(
  id: number,
  uploadKey: string,
): Promise<Place> {
  return unwrap(
    await apiPut<Place, PlaceImageErrorCode>(`places/${id}/image`, {
      json: { upload_key: uploadKey },
    }),
  );
}
