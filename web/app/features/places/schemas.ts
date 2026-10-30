import { z } from "zod";

import { parseNumber, type AttributeValues } from "./attribute-values";

// -----------------------------------------------------------------------------
// Place form
// -----------------------------------------------------------------------------

export function validCoordinate(text: string, limit: number): boolean {
  const value = parseNumber(text);
  return Number.isFinite(value) && Math.abs(value) <= limit;
}

// Indonesia's bounding box, mirroring the backend (internal/module/place/constant.go).
const INDONESIA = { minLat: -11, maxLat: 6, minLng: 95, maxLng: 141 };

/** Whether a coordinate lies within [min, max]; text that is not a number is left to `validCoordinate`. */
function withinBounds(text: string, min: number, max: number): boolean {
  const value = parseNumber(text);
  return !Number.isFinite(value) || (value >= min && value <= max);
}

export const placeSchema = z.object({
  subDistrictId: z.string().min(1, "Pilih kecamatan."),
  categoryId: z.string().min(1, "Pilih kategori."),
  name: z
    .string()
    .trim()
    .min(1, "Nama tempat wajib diisi.")
    .max(150, "Nama tempat maksimal 150 karakter."),
  lat: z
    .string()
    .refine((v) => validCoordinate(v, 90), "Isi lintang, atau ketuk peta.")
    .refine(
      (v) => withinBounds(v, INDONESIA.minLat, INDONESIA.maxLat),
      "Lokasi harus di Indonesia: lintang antara -11 dan 6.",
    ),
  lng: z
    .string()
    .refine((v) => validCoordinate(v, 180), "Isi bujur, atau ketuk peta.")
    .refine(
      (v) => withinBounds(v, INDONESIA.minLng, INDONESIA.maxLng),
      "Lokasi harus di Indonesia: bujur antara 95 dan 141.",
    ),
  address: z.string(),
  operatingHours: z.string(),
  priceRange: z.string(),
  description: z.string(),
  attributes: z.custom<AttributeValues>(),
});
