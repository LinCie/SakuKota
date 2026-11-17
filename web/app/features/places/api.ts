import { apiDelete, apiGet, apiPost, apiPut, unwrap } from "~/shared/api";
import type { MediaUpload } from "~/shared/image-upload";
import type { CategoryDetail, GeoPoint, Place, PlacePage } from "./types";

// Every function here calls one backend route (see internal/router/router.go),
// returns the response data, and throws an `ApiError` when the backend rejects
// the request.

export type GetPlacesParams = {
  /** Filters places to one sub-district, matching the `sub_district_id` query param. */
  subDistrictId?: number;
  /** Keeps places in any of these categories, matching the `category_id` query param. */
  categoryIds?: number[];
  /** Case-insensitive search term, matching the `q` query param. */
  q?: string;
  /** Keeps only the first places, matching the `limit` query param. The server caps it at 100. */
  limit?: number;
  /** Skips the first places before `limit` applies, matching the `offset` query param. */
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

export async function getPlaces(
  params: GetPlacesParams = {},
): Promise<PlacePage> {
  const { subDistrictId, categoryIds, q, limit, offset, bbox } = params;

  // Only the filters that are set go into the URL.
  const searchParams = new URLSearchParams();
  if (subDistrictId !== undefined) {
    searchParams.set("sub_district_id", String(subDistrictId));
  }
  if (categoryIds?.length) {
    searchParams.set("category_id", categoryIds.join(","));
  }
  if (q) searchParams.set("q", q);
  if (limit !== undefined) searchParams.set("limit", String(limit));
  if (offset) searchParams.set("offset", String(offset));
  if (bbox) searchParams.set("bbox", bbox);

  return unwrap(await apiGet<PlacePage>("places", { searchParams }));
}

export async function getPlace(id: number): Promise<Place> {
  return unwrap(await apiGet<Place>(`places/${id}`));
}

export async function getCategories(): Promise<CategoryDetail[]> {
  return unwrap(await apiGet<CategoryDetail[]>("places/categories"));
}

/** Admin only. */
export async function createPlace(body: PlaceRequest): Promise<Place> {
  return unwrap(await apiPost<Place>("places", { json: body }));
}

/** Admin only. */
export async function updatePlace(
  id: number,
  body: PlaceRequest,
): Promise<Place> {
  return unwrap(await apiPut<Place>(`places/${id}`, { json: body }));
}

/** Admin only. */
export async function deletePlace(id: number): Promise<void> {
  unwrap(await apiDelete<void>(`places/${id}`));
}

/** Admin only. Presigns an upload of a new image of the place. */
export async function requestPlaceMediaUpload(
  id: number,
  contentType: string,
): Promise<MediaUpload> {
  return unwrap(
    await apiPost<MediaUpload>(`places/${id}/medias/upload-url`, {
      json: { content_type: contentType },
    }),
  );
}

/** Admin only. Adds the uploaded image to the place's medias. */
export async function addPlaceMedia(
  id: number,
  uploadKey: string,
): Promise<Place> {
  return unwrap(
    await apiPost<Place>(`places/${id}/medias`, {
      json: { upload_key: uploadKey },
    }),
  );
}

/** Admin only. */
export async function deletePlaceMedia(
  id: number,
  mediaId: number,
): Promise<void> {
  unwrap(await apiDelete<void>(`places/${id}/medias/${mediaId}`));
}
