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
  static_map_image_url: string | null;
};
