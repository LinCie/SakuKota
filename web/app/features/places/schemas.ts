import { z } from "zod";

import { parseNumber, type AttributeValues } from "./attribute-values";

// -----------------------------------------------------------------------------
// Place form
// -----------------------------------------------------------------------------

export function validCoordinate(text: string, limit: number): boolean {
  const value = parseNumber(text);
  return Number.isFinite(value) && Math.abs(value) <= limit;
}

export const placeSchema = z.object({
  subDistrictId: z.string().min(1, "Pilih kecamatan."),
  categoryId: z.string().min(1, "Pilih kategori."),
  name: z.string().trim().min(1, "Nama tempat wajib diisi."),
  lat: z
    .string()
    .refine((v) => validCoordinate(v, 90), "Isi lintang, atau ketuk peta."),
  lng: z
    .string()
    .refine((v) => validCoordinate(v, 180), "Isi bujur, atau ketuk peta."),
  address: z.string(),
  operatingHours: z.string(),
  priceRange: z.string(),
  description: z.string(),
  attributes: z.custom<AttributeValues>(),
});
