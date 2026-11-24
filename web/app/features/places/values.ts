import type { SubDistrict } from "~/features/subdistricts/types";

import { emptyHours, type HoursValues } from "./hours";
import type { CategoryField } from "./types";

// The place form's values, as typed: everything is text until it is submitted.

/**
 * What a custom field holds while editing: text for most types (numbers stay
 * text until submit), "true" / "false" / "" for yes-no, and two texts for a range.
 */
export type AttributeValue = string | { min: string; max: string };
export type AttributeValues = Record<string, AttributeValue | undefined>;

export type PlaceValues = {
  subDistrictId: string;
  categoryId: string;
  name: string;
  lat: string;
  lng: string;
  address: string;
  /** The seven days, Monday first. */
  hours: HoursValues;
  /** The lowest and highest price in rupiah; one filled bound is a single price. */
  priceMin: string;
  priceMax: string;
  description: string;
  attributes: AttributeValues;
};

export function emptyPlaceValues(subDistricts: SubDistrict[]): PlaceValues {
  return {
    // A single kecamatan is the only choice, so pick it.
    subDistrictId: subDistricts.length === 1 ? String(subDistricts[0].id) : "",
    categoryId: "",
    name: "",
    lat: "",
    lng: "",
    address: "",
    hours: emptyHours(),
    priceMin: "",
    priceMax: "",
    description: "",
    attributes: {},
  };
}

export function textOf(value: AttributeValue | undefined): string {
  return typeof value === "string" ? value : "";
}

export function rangeOf(value: AttributeValue | undefined) {
  return typeof value === "object" ? value : { min: "", max: "" };
}

/** Whether a custom field was left empty. */
export function isBlank(
  value: AttributeValue | undefined,
  field: CategoryField,
) {
  if (field.data_type === "range") {
    const { min, max } = rangeOf(value);
    return min.trim() === "" && max.trim() === "";
  }
  return textOf(value).trim() === "";
}

// Plain decimals only: `Number` alone would also accept "Infinity", "1e3" and "0x10".
const decimal = /^[+-]?(\d+([.,]\d*)?|[.,]\d+)$/;

/** Numbers typed with a decimal comma ("1,5") are common on Indonesian keyboards. */
export function parseNumber(text: string): number {
  const trimmed = text.trim();
  if (!decimal.test(trimmed)) return NaN;
  const value = Number(trimmed.replace(",", "."));
  // Hundreds of digits overflow to Infinity.
  return Number.isFinite(value) ? value : NaN;
}
