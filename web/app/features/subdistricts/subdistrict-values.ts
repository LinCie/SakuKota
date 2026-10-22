import { orNull } from "~/shared/text";

import type { CreateSubDistrictRequest } from "./api";
import type { SubDistrict } from "./types";

/** The kecamatan form's values, as typed. */
export type SubDistrictValues = {
  name: string;
  slug: string;
  floodHistory: string;
  safetyInfo: string;
  dailySpending: string;
  mapImageUrl: string;
};

export const emptySubDistrictValues: SubDistrictValues = {
  name: "",
  slug: "",
  floodHistory: "",
  safetyInfo: "",
  dailySpending: "",
  mapImageUrl: "",
};

export function subDistrictToValues(
  subDistrict: SubDistrict,
): SubDistrictValues {
  return {
    name: subDistrict.name,
    slug: subDistrict.slug,
    floodHistory: subDistrict.flood_history ?? "",
    safetyInfo: subDistrict.safety_crime_information ?? "",
    dailySpending:
      subDistrict.estimated_daily_spending === null
        ? ""
        : String(subDistrict.estimated_daily_spending),
    mapImageUrl: subDistrict.static_map_image_url ?? "",
  };
}

/** Digits only; dots and spaces are accepted as thousands separators. */
function parseRupiah(text: string): number | null {
  const digits = text.replace(/[.\s]/g, "");
  return digits === "" ? null : Number(digits);
}

/** The request body for creating or replacing a kecamatan. */
export function toSubDistrictRequest(
  values: SubDistrictValues,
): CreateSubDistrictRequest {
  return {
    name: values.name.trim(),
    slug: values.slug.trim(),
    flood_history: orNull(values.floodHistory),
    safety_crime_information: orNull(values.safetyInfo),
    estimated_daily_spending: parseRupiah(values.dailySpending),
    static_map_image_url: orNull(values.mapImageUrl),
  };
}
