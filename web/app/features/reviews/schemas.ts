import { z } from "zod";

import { maxMedias } from "~/shared/image-upload";

// Mirrors the backend's rules (internal/module/reviews): a rating of 1 to 5, a
// comment of at most 1000 characters once trimmed, and at most 10 files.

export const reviewSchema = z.object({
  rating: z.number().int().min(1, "Pilih jumlah bintang.").max(5),
  comment: z
    .string()
    // Counted by code point, as the backend counts characters.
    .refine(
      (value) => [...value.trim()].length <= 1000,
      "Komentar maksimal 1000 karakter.",
    ),
  files: z
    .array(z.instanceof(File))
    .max(maxMedias, `Maksimal ${maxMedias} file.`),
});

/** The review form's values, as typed. A rating of 0 means none is picked. */
export type ReviewValues = {
  rating: number;
  comment: string;
  /** Picked files, added to the review once it is saved. */
  files: File[];
};
