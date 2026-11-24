import { formatRupiah } from "~/shared/format";

import type { PriceRange } from "./types";
import { parseNumber } from "./values";

// A place's price range: how it reads, and the form's min and max texts. The
// backend's rules (domain.NormalizePlaceFields) are mirrored by `priceError`.

/** The highest price the backend accepts, in rupiah. */
export const MAX_PRICE = 1_000_000_000;

/**
 * The range as people read it: "Rp 9.000–Rp 12.000/kg", "Rp 3.500" for a single
 * price, "Gratis" for 0 and "Gratis–Rp 50.000" from 0. Null when the price is unknown.
 */
export function formatPrice(
  range: PriceRange | null,
  unit: string | null,
): string | null {
  if (!range) return null;

  const amount = (value: number) =>
    value === 0 ? "Gratis" : formatRupiah(value);
  const text =
    range.min === range.max
      ? amount(range.min)
      : `${amount(range.min)}–${amount(range.max)}`;

  return unit ? `${text}/${unit}` : text;
}

/** The range as the form's min and max texts; empty when unknown. */
export function priceToValues(range: PriceRange | null) {
  return {
    priceMin: range ? String(range.min) : "",
    priceMax: range ? String(range.max) : "",
  };
}

/**
 * The request's range from the form's texts. A single filled bound is both
 * bounds, so one price needs one input. Null when both are blank, or when the
 * category has no prices, whatever was typed before it was chosen.
 */
export function priceToRequest(
  min: string,
  max: string,
  hasPrice: boolean,
): PriceRange | null {
  if (!hasPrice) return null;

  const low = min.trim() === "" ? undefined : parseNumber(min);
  const high = max.trim() === "" ? undefined : parseNumber(max);
  const bound = low ?? high;
  if (bound === undefined) return null;

  return { min: low ?? bound, max: high ?? bound };
}

/**
 * Whether text is a whole number of rupiah within the backend's bounds. Digits
 * only: "15.000" would otherwise read as 15 with a decimal point.
 */
function validAmount(text: string): boolean {
  const trimmed = text.trim();
  return /^\d+$/.test(trimmed) && Number(trimmed) <= MAX_PRICE;
}

/**
 * The problem with the typed min and max, or undefined when they are valid or
 * both blank. Error messages are keyed by the input they belong to.
 */
export function priceError(
  min: string,
  max: string,
): { field: "priceMin" | "priceMax"; message: string } | undefined {
  const inputs = [
    { field: "priceMin", text: min },
    { field: "priceMax", text: max },
  ] as const;

  for (const { field, text } of inputs) {
    if (text.trim() !== "" && !validAmount(text)) {
      return {
        field,
        message: `Isi rupiah bulat, antara 0 dan ${MAX_PRICE.toLocaleString("id-ID")}.`,
      };
    }
  }

  if (
    min.trim() !== "" &&
    max.trim() !== "" &&
    parseNumber(min) > parseNumber(max)
  ) {
    return {
      field: "priceMin",
      message: "Harga terendah tidak boleh melebihi harga tertinggi.",
    };
  }

  return undefined;
}
