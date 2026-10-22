import { getSubDistricts } from "~/features/subdistricts/api";
import { unwrap } from "~/shared/api";

import { getCategories, getPlaces } from "./api";

/** Everything the places pages need: the places, their kecamatan, and categories with fields. */
export async function loadPlaceData() {
  const [categories, subDistricts, places] = await Promise.all([
    getCategories(),
    getSubDistricts(),
    getPlaces(),
  ]);

  return {
    categories: unwrap(categories),
    subDistricts: unwrap(subDistricts),
    places: unwrap(places),
  };
}
