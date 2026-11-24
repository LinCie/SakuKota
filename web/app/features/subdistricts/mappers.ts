import { orNull } from "~/shared/text";

import type { SubDistrictRequest } from "./api";
import type { SubDistrict } from "./types";

/** The kecamatan form's values, as typed: everything is text until it is submitted. */
export type SubDistrictValues = {
  name: string;
  slug: string;
  floodHistory: string;
  safetyInformation: string;
  dailySpending: string;
};

export const emptySubDistrictValues: SubDistrictValues = {
  name: "",
  slug: "",
  floodHistory: "",
  safetyInformation: "",
  dailySpending: "",
};

/** Rupiah with up to two decimals and no thousands separators, so "50.000" is not read as 50. */
const spending = /^\d+([.,]\d{1,2})?$/;

/** The typed daily spending as a number, or NaN if it is not one. */
export function parseSpending(text: string): number {
  const trimmed = text.trim();
  return spending.test(trimmed) ? Number(trimmed.replace(",", ".")) : NaN;
}

export function subDistrictToValues(sd: SubDistrict): SubDistrictValues {
  return {
    name: sd.name,
    slug: sd.slug,
    floodHistory: sd.flood_history ?? "",
    safetyInformation: sd.safety_crime_information ?? "",
    dailySpending:
      sd.estimated_daily_spending === null
        ? ""
        : String(sd.estimated_daily_spending),
  };
}

/** The request body for creating or replacing a kecamatan; it sends every field. */
export function toSubDistrictRequest(
  values: SubDistrictValues,
): SubDistrictRequest {
  const dailySpending = values.dailySpending.trim();
  return {
    name: values.name.trim(),
    slug: values.slug.trim(),
    flood_history: orNull(values.floodHistory),
    safety_crime_information: orNull(values.safetyInformation),
    estimated_daily_spending:
      dailySpending === "" ? null : parseSpending(dailySpending),
  };
}
