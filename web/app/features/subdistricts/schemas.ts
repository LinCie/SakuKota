import { z } from "zod";

export const subDistrictSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Nama kecamatan wajib diisi.")
    .max(100, "Nama kecamatan maksimal 100 karakter."),
  slug: z
    .string()
    .trim()
    .min(1, "Slug wajib diisi.")
    .max(100, "Slug maksimal 100 karakter.")
    .regex(
      /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
      "Slug hanya huruf kecil, angka, dan tanda hubung.",
    ),
  floodHistory: z.string(),
  safetyInfo: z.string(),
  dailySpending: z
    .string()
    .refine(
      (v) => /^\d*$/.test(v.replace(/[.\s]/g, "")),
      "Isi dengan angka saja, contoh: 75000.",
    )
    // The backend's DECIMAL(10,2) column holds at most 99,999,999.99.
    .refine(
      (v) => !(Number(v.replace(/[.\s]/g, "")) > 99_999_999),
      "Maksimal Rp99.999.999.",
    ),
});
