import type { SubDistrict } from "~/features/subdistricts/types";
import { orNull } from "~/shared/text";

import type { PlaceRequest } from "./api";
import { isRange } from "./attribute-format";
import {
  buildAttributes,
  parseNumber,
  type AttributeValues,
} from "./attribute-values";
import type { CategoryField, Place } from "./types";

/** The place form's values, as typed. */
export type PlaceValues = {
  subDistrictId: string;
  categoryId: string;
  name: string;
  lat: string;
  lng: string;
  address: string;
  operatingHours: string;
  priceRange: string;
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
    operatingHours: "",
    priceRange: "",
    description: "",
    attributes: {},
  };
}

/** A place as form values, reading its custom fields by their data type. */
export function placeToValues(
  place: Place,
  fields: CategoryField[],
): PlaceValues {
  const coordinates = place.location?.coordinates;
  return {
    subDistrictId: String(place.sub_district_id),
    categoryId: String(place.category.id),
    name: place.name,
    // GeoJSON orders coordinates as [longitude, latitude].
    lat: coordinates ? String(coordinates[1]) : "",
    lng: coordinates ? String(coordinates[0]) : "",
    address: place.address ?? "",
    operatingHours: place.operating_hours ?? "",
    priceRange: place.estimated_price_range ?? "",
    description: place.short_description ?? "",
    attributes: attributesToValues(fields, place.attributes),
  };
}

/** Custom field values as form values, read by each field's data type. */
export function attributesToValues(
  fields: CategoryField[],
  attributes: Record<string, unknown>,
): AttributeValues {
  const values: AttributeValues = {};

  for (const field of fields) {
    const value = attributes[field.key];
    switch (field.data_type) {
      case "range":
        if (isRange(value)) {
          values[field.key] = {
            min: String(value.min),
            max: String(value.max),
          };
        }
        break;
      case "boolean":
        if (typeof value === "boolean") {
          values[field.key] = value ? "true" : "false";
        }
        break;
      case "date":
        // Date inputs take YYYY-MM-DD.
        if (typeof value === "string") values[field.key] = value.slice(0, 10);
        break;
      default:
        if (typeof value === "string" || typeof value === "number") {
          values[field.key] = String(value);
        }
    }
  }

  return values;
}

/** The request body for creating or replacing a place. */
export function toPlaceRequest(
  values: PlaceValues,
  fields: CategoryField[],
): PlaceRequest {
  return {
    sub_district_id: Number(values.subDistrictId),
    category_id: Number(values.categoryId),
    name: values.name.trim(),
    address: orNull(values.address),
    location: {
      type: "Point",
      coordinates: [parseNumber(values.lng), parseNumber(values.lat)],
    },
    operating_hours: orNull(values.operatingHours),
    estimated_price_range: orNull(values.priceRange),
    short_description: orNull(values.description),
    attributes: buildAttributes(fields, values.attributes),
  };
}
