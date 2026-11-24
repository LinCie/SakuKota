import { z } from "zod";

import { checkHoursAndPrice, placeShape } from "~/features/places/schemas";

// Mirrors the backend's rules (internal/module/suggestion/service.go): the
// place's own rules, plus a note and a rejection reason of at most 1000
// characters once trimmed, counted by code point as the backend counts them.

function atMost1000(value: string) {
  return [...value.trim()].length <= 1000;
}

export const suggestionSchema = z
  .object({
    ...placeShape,
    note: z.string().refine(atMost1000, "Catatan maksimal 1000 karakter."),
  })
  .superRefine(checkHoursAndPrice);

export const rejectSchema = z.object({
  reason: z.string().refine(atMost1000, "Alasan maksimal 1000 karakter."),
});
