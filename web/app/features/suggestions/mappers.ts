import { attributesToValues, toPlaceRequest } from "~/features/places/mappers";
import type { CategoryField } from "~/features/places/types";
import type { PlaceValues } from "~/features/places/values";
import { orNull } from "~/shared/text";

import type { CreateSuggestionRequest } from "./api";
import type { Suggestion } from "./types";
import type { SuggestionValues } from "./values";

/** The request body for suggesting a place. */
export function toSuggestionRequest(
  values: SuggestionValues,
  fields: CategoryField[],
): CreateSuggestionRequest {
  return { ...toPlaceRequest(values, fields), note: orNull(values.note) };
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
    operatingHours: suggestion.operating_hours ?? "",
    priceRange: suggestion.estimated_price_range ?? "",
    description: suggestion.short_description ?? "",
    attributes: attributesToValues(fields, suggestion.attributes),
  };
}
