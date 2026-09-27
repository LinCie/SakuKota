// -----------------------------------------------------------------------------
// Types (see internal/module/place/dto.go and internal/postgis/point.go)
// -----------------------------------------------------------------------------

/** A PostGIS point, serialized by the backend as a GeoJSON Point. */
export type GeoPoint = {
  type: "Point";
  /** [longitude, latitude] */
  coordinates: [number, number];
};

export type Place = {
  id: number;
  sub_district_id: number;
  name: string;
  address: string | null;
  location: GeoPoint | null;
  operating_hours: string | null;
  estimated_price_range: string | null;
  short_description: string | null;
  categories: Category[];
};

export type Category = {
  id: number;
  name: string;
  slug: string;
};
