import type { CreateCategoryFieldRequest } from "~/features/places/api";

import { parseOptions, type CategoryFieldValues } from "./schemas";

export const emptyCategoryFieldValues: CategoryFieldValues = {
  label: "",
  key: "",
  dataType: "text",
  required: false,
  options: "",
};

/** The request body for a new field. `options` only applies to select fields. */
export function toCategoryFieldRequest(
  values: CategoryFieldValues,
): CreateCategoryFieldRequest {
  return {
    key: values.key.trim(),
    label: values.label.trim(),
    data_type: values.dataType,
    required: values.required,
    options:
      values.dataType === "select" ? parseOptions(values.options) : undefined,
  };
}
