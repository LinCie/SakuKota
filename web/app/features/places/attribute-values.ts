import type { CategoryField } from "./types";

// -----------------------------------------------------------------------------
// Custom field values, as typed into the form
// -----------------------------------------------------------------------------

/**
 * What a custom field holds while editing: text for most types (numbers stay
 * text until submit), "true" / "false" / "" for yes-no, and two texts for a range.
 */
export type AttributeValue = string | { min: string; max: string };
export type AttributeValues = Record<string, AttributeValue | undefined>;

export function textOf(value: AttributeValue | undefined): string {
  return typeof value === "string" ? value : "";
}

export function rangeOf(value: AttributeValue | undefined) {
  return typeof value === "object" ? value : { min: "", max: "" };
}

/** Numbers typed with a decimal comma ("1,5") are common on Indonesian keyboards. */
export function parseNumber(text: string): number {
  const trimmed = text.trim().replace(",", ".");
  return trimmed === "" ? NaN : Number(trimmed);
}

function isBlank(value: AttributeValue | undefined, field: CategoryField) {
  if (field.data_type === "range") {
    const { min, max } = rangeOf(value);
    return min.trim() === "" && max.trim() === "";
  }
  return textOf(value).trim() === "";
}

/** Error messages by field key; empty when every custom field is valid. */
export function attributeErrors(
  fields: CategoryField[],
  values: AttributeValues,
): Record<string, string> {
  const errors: Record<string, string> = {};

  for (const field of fields) {
    const value = values[field.key];

    if (isBlank(value, field)) {
      if (field.required) errors[field.key] = "Wajib diisi.";
      continue;
    }

    if (
      field.data_type === "number" &&
      Number.isNaN(parseNumber(textOf(value)))
    ) {
      errors[field.key] = "Isi dengan angka.";
    }

    if (field.data_type === "range") {
      const range = rangeOf(value);
      const min = parseNumber(range.min);
      const max = parseNumber(range.max);
      if (Number.isNaN(min) || Number.isNaN(max)) {
        errors[field.key] = "Isi batas bawah dan batas atas dengan angka.";
      } else if (min > max) {
        errors[field.key] = "Batas bawah tidak boleh melebihi batas atas.";
      }
    }
  }

  return errors;
}

/** The request's `attributes`: filled custom fields only, typed per field. */
export function buildAttributes(
  fields: CategoryField[],
  values: AttributeValues,
): Record<string, unknown> {
  const attributes: Record<string, unknown> = {};

  for (const field of fields) {
    const value = values[field.key];
    if (isBlank(value, field)) continue;

    switch (field.data_type) {
      case "number":
        attributes[field.key] = parseNumber(textOf(value));
        break;
      case "boolean":
        attributes[field.key] = textOf(value) === "true";
        break;
      case "range": {
        const range = rangeOf(value);
        attributes[field.key] = {
          min: parseNumber(range.min),
          max: parseNumber(range.max),
        };
        break;
      }
      default:
        attributes[field.key] = textOf(value).trim();
    }
  }

  return attributes;
}
