import { z } from "zod";

export const subDistrictSchema = z.object({
  name: z.string().trim().min(1, "Nama kecamatan wajib diisi."),
  slug: z
    .string()
    .trim()
    .min(1, "Slug wajib diisi.")
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
    ),
  mapImageUrl: z
    .string()
    .trim()
    .refine(
      (v) => v === "" || /^https?:\/\/\S+$/.test(v),
      "Isi alamat gambar yang diawali http:// atau https://.",
    ),
});
