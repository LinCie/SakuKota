import type { Media } from "~/shared/image-upload";

// Types (see internal/module/place/dto.go and internal/pkg/postgis/point.go)

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
  /** Periods the place is open, sorted by day and opening time; null if unknown. */
  operating_hours: OperatingPeriod[] | null;
  /** Null if unknown, and always for a category without prices. */
  estimated_price_range: PriceRange | null;
  short_description: string | null;
  /** Values of the category's custom fields, keyed by `CategoryField.key`. */
  attributes: Record<string, unknown>;
  /** Mean star rating of the place's reviews; null before the first review. */
  rating_average: number | null;
  rating_count: number;
  /** The place's images in position order; empty if it has none. */
  medias: Media[];
  category: Category;
};

/** One stretch of a weekday during which a place is open. */
export type OperatingPeriod = {
  /** 1 is Monday, 7 is Sunday. */
  day: number;
  /** Local time as HH:MM, before `close`. */
  open: string;
  /** Local time as HH:MM, or 24:00 for midnight. */
  close: string;
};

/** A price range in whole rupiah; a single price has equal bounds. */
export type PriceRange = {
  min: number;
  max: number;
};

/** The places matching a filter, with how many match before the offset and limit. */
export type PlacePage = {
  places: Place[];
  total: number;
};

/**
 * The palette keys a category can be colored with. Categories are defined by
 * the backend's migrations, whose `color` column only allows these.
 */
export type CategoryColor = "chili" | "plum" | "cat" | "sign" | "board";

export type Category = {
  id: number;
  name: string;
  slug: string;
  /** The single character that marks the category on a sign. */
  code: string;
  /** Short name to display. */
  label: string;
  color: CategoryColor;
  /** Whether places of the category have a price. A mosque has none. */
  has_price: boolean;
  /** What the price is per, such as "kg"; null for the whole item or service. */
  price_unit: string | null;
};

/** Whether an attribute value is a range, as the backend stores a `range` field. */
export function isRange(value: unknown): value is { min: number; max: number } {
  return (
    typeof value === "object" &&
    value !== null &&
    "min" in value &&
    "max" in value &&
    typeof value.min === "number" &&
    typeof value.max === "number"
  );
}

/** A category with the custom fields its places carry. */
export type CategoryDetail = Category & {
  fields: CategoryField[];
};

/**
 * Validated by the backend (internal/module/place/service.go). New types may
 * arrive before the frontend knows them, so they must still render.
 */
export type FieldDataType =
  | "text"
  | "number"
  | "boolean"
  | "select"
  | "range"
  | "date"
  // Any other type, while keeping autocomplete for the known ones.
  | (string & {});

export type CategoryField = {
  id: number;
  category_id: number;
  key: string;
  label: string;
  data_type: FieldDataType;
  required: boolean;
  /** Allowed values of a select field; null for other types. */
  options: string[] | null;
  /** 1-based position among the category's fields, which arrive sorted by it. */
  sort_order: number;
};
