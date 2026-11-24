import type { ReactNode } from "react";

import { orMissing } from "~/shared/components/missing-slot";

import { formatHours } from "../hours";
import { formatPrice } from "../price";
import type { Category, OperatingPeriod, PriceRange } from "../types";

/** The place's hours, one line per run of days, or a slot saying they are missing. */
export function hoursValue(hours: OperatingPeriod[] | null): ReactNode {
  const lines = formatHours(hours);
  return orMissing(
    lines.length > 0 ? (
      <span className="flex flex-col tabular-nums">
        {lines.map((line) => (
          <span key={line}>{line}</span>
        ))}
      </span>
    ) : null,
    "Jam buka tidak tersedia",
  );
}

/**
 * The place's price range, or a slot saying it is missing. A category without
 * prices has nothing to show; callers leave the price out for it.
 */
export function priceValue(
  range: PriceRange | null,
  category: Pick<Category, "price_unit">,
): ReactNode {
  return orMissing(
    formatPrice(range, category.price_unit),
    "Harga tidak tersedia",
  );
}
