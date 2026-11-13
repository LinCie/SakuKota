import type { Media } from "~/shared/image-upload";

// -----------------------------------------------------------------------------
// Types (see internal/module/subdistrict/dto.go)
// -----------------------------------------------------------------------------

export type SubDistrict = {
  id: number;
  name: string;
  slug: string;
  flood_history: string | null;
  safety_crime_information: string | null;
  estimated_daily_spending: number | null;
  /** The sub-district's images in position order; empty if it has none. */
  medias: Media[];
  /** Public URL of the sub-district's static map; null if it has none. */
  static_map_image_url: string | null;
};
