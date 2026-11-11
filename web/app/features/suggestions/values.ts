import { emptyPlaceValues, type PlaceValues } from "~/features/places/values";
import type { SubDistrict } from "~/features/subdistricts/types";

/** The suggestion form's values: the place as typed, plus a note to the reviewer. */
export type SuggestionValues = PlaceValues & { note: string };

export function emptySuggestionValues(
  subDistricts: SubDistrict[],
): SuggestionValues {
  return { ...emptyPlaceValues(subDistricts), note: "" };
}
