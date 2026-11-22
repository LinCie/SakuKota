import type { Media } from "~/shared/image-upload";

// Types (see internal/module/subdistricts/dto.go)

/** A GeoJSON MultiPolygon: polygons of rings of [longitude, latitude]. */
export type Boundary = {
  type: "MultiPolygon";
  coordinates: number[][][][];
};

export type SubDistrict = {
  id: number;
  name: string;
  slug: string;
  flood_history: string | null;
  safety_crime_information: string | null;
  estimated_daily_spending: number | null;
  /** The sub-district's images in position order; empty if it has none. */
  medias: Media[];
  /** The sub-district's area; null if it has none. */
  boundary: Boundary | null;
};
