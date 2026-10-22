import { Check, X } from "lucide-react";
import type { ReactNode } from "react";

import { orMissing } from "~/shared/components/missing-slot";
import { formatDate, formatNumber } from "~/shared/format";

import type { CategoryField, Place } from "./types";

export function isRange(value: unknown): value is { min: number; max: number } {
  return (
    typeof value === "object" &&
    value !== null &&
    "min" in value &&
    "max" in value &&
    typeof value.min === "number" &&
    typeof value.max === "number"
  );
}

/**
 * Renders a custom field value by its data type, or null when the value is
 * missing or doesn't match the type.
 */
export function formatAttribute(
  field: CategoryField,
  value: unknown,
): ReactNode {
  switch (field.data_type) {
    case "text":
    case "select":
      return typeof value === "string" && value.trim() !== "" ? value : null;
    case "number":
      return typeof value === "number" ? formatNumber(value) : null;
    case "boolean": {
      if (typeof value !== "boolean") return null;
      const Icon = value ? Check : X;
      return (
        // The word sets the baseline so the row's leader lines up with it.
        <span className="inline-flex items-baseline gap-1">
          <Icon
            aria-hidden="true"
            className="size-4 self-center"
            strokeWidth={2.5}
          />
          {value ? "Ya" : "Tidak"}
        </span>
      );
    }
    case "range":
      if (!isRange(value)) return null;
      return value.min === value.max
        ? formatNumber(value.min)
        : `${formatNumber(value.min)}–${formatNumber(value.max)}`;
    case "date": {
      const date = typeof value === "string" ? new Date(value) : null;
      return date && !Number.isNaN(date.getTime()) ? formatDate(date) : null;
    }
    default:
      // A type this frontend doesn't know yet: show plain values as-is.
      return typeof value === "string" || typeof value === "number"
        ? String(value)
        : null;
  }
}

/** A place's value for one custom field, or a drawn slot when it is missing. */
export function attributeValue(field: CategoryField, place: Place): ReactNode {
  return orMissing(formatAttribute(field, place.attributes[field.key]));
}
