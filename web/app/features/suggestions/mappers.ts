import { hoursToValues } from "~/features/places/hours";
import { attributesToValues, toPlaceRequest } from "~/features/places/mappers";
import { priceToValues } from "~/features/places/price";
import type { CategoryDetail, CategoryField } from "~/features/places/types";
import { emptyPlaceValues, type PlaceValues } from "~/features/places/values";
import type { SubDistrict } from "~/features/subdistricts/types";
import { orNull } from "~/shared/text";

import type { CreateSuggestionRequest } from "./api";
import type { Suggestion } from "./types";

/** The suggestion form's values: the place as typed, plus a note to the reviewer. */
export type SuggestionValues = PlaceValues & { note: string };

export function emptySuggestionValues(
  subDistricts: SubDistrict[],
): SuggestionValues {
  return { ...emptyPlaceValues(subDistricts), note: "" };
}

/** The request body for suggesting a place. */
export function toSuggestionRequest(
  values: SuggestionValues,
  category: Pick<CategoryDetail, "fields" | "has_price">,
): CreateSuggestionRequest {
  return { ...toPlaceRequest(values, category), note: orNull(values.note) };
}

/** A suggestion as place form values, the starting point for approving it. */
export function suggestionToPlaceValues(
  suggestion: Suggestion,
  fields: CategoryField[],
): PlaceValues {
  const coordinates = suggestion.location?.coordinates;
  return {
    subDistrictId: String(suggestion.sub_district_id),
    categoryId: String(suggestion.category_id),
    name: suggestion.name,
    // GeoJSON orders coordinates as [longitude, latitude].
    lat: coordinates ? String(coordinates[1]) : "",
    lng: coordinates ? String(coordinates[0]) : "",
    address: suggestion.address ?? "",
    hours: hoursToValues(suggestion.operating_hours),
    ...priceToValues(suggestion.estimated_price_range),
    description: suggestion.short_description ?? "",
    attributes: attributesToValues(fields, suggestion.attributes),
  };
}
