import { z } from "zod";

import { REPORT_REASONS } from "./types";

// Mirrors the backend's rules (internal/module/reports/service.go): one of the
// reasons, and a note of at most 1000 characters once trimmed, counted by code
// point as the backend counts them.

export const reportSchema = z.object({
  reason: z.enum(REPORT_REASONS, "Pilih alasan."),
  note: z
    .string()
    .refine(
      (value) => [...value.trim()].length <= 1000,
      "Catatan maksimal 1000 karakter.",
    ),
});

/** The report form's values, as typed. An empty reason means none is picked. */
export type ReportValues = { reason: string; note: string };
