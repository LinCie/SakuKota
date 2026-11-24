import { z } from "zod";

import { parseSpending } from "./mappers";

// Mirrors the backend's rules (internal/module/subdistricts/service.go).

const MAX_SPENDING = 99_999_999.99;

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
    .max(100, "Slug maksimal 100 karakter."),
  floodHistory: z.string(),
  safetyInformation: z.string(),
  dailySpending: z.string().refine((text) => {
    if (text.trim() === "") return true;
    const value = parseSpending(text);
    return !Number.isNaN(value) && value <= MAX_SPENDING;
  }, "Isi angka 0 sampai 99999999,99 tanpa titik ribuan, contoh 75000."),
});
