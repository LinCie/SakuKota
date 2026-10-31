import { z } from "zod";

// Mirrors the backend's rules (internal/module/review/service.go): a rating of
// 1 to 5, and a comment of at most 1000 characters once trimmed.

export const reviewSchema = z.object({
  rating: z.number().int().min(1, "Pilih jumlah bintang.").max(5),
  comment: z
    .string()
    // Counted by code point, as the backend counts characters.
    .refine(
      (value) => [...value.trim()].length <= 1000,
      "Komentar maksimal 1000 karakter.",
    ),
});

/** The review form's values, as typed. A rating of 0 means none is picked. */
export type ReviewValues = {
  rating: number;
  comment: string;
};
