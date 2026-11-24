import {
  Banknote,
  Building2,
  Church,
  Droplets,
  HeartPulse,
  Hospital,
  type LucideIcon,
  MapPin,
  Mosque,
  Pill,
  Printer,
  Scissors,
  ShieldCheck,
  Stethoscope,
  Store,
  TrainFront,
  UtensilsCrossed,
  WashingMachine,
} from "lucide-react";

import { cn } from "~/shared/cn";
import type { Category, CategoryColor } from "../types";

type CategoryStyle = {
  icon: LucideIcon;
  label: string;
  /** Soft tint behind its deep tone: category tiles (see DESIGN.md). */
  tint: string;
  /** Solid tint with its icon colour: map pins. */
  pin: string;
};

// The tints a category's `color` selects (see DESIGN.md). Class strings are complete literals
// so Tailwind can see them, including inside the map's marker HTML.
const CATEGORY_COLORS: Record<
  CategoryColor,
  Pick<CategoryStyle, "tint" | "pin">
> = {
  chili: {
    tint: "bg-sambal-soft text-sambal-deep",
    pin: "bg-sambal text-card",
  },
  plum: {
    tint: "bg-ubi-soft text-ubi-deep",
    pin: "bg-ubi text-card",
  },
  cat: {
    tint: "bg-telang-soft text-telang-deep",
    pin: "bg-telang text-card",
  },
  sign: {
    tint: "bg-kunyit-soft text-kunyit-deep",
    pin: "bg-kunyit text-kunyit-deep",
  },
  board: {
    tint: "bg-pandan-soft text-pandan-deep",
    pin: "bg-pandan text-card",
  },
};

// The icon a category shows on its pin, tile, and badge, keyed by the migrations' `slug`.
// Categories without an entry fall back to a generic map pin.
const CATEGORY_ICONS: Record<string, LucideIcon> = {
  "affordable-food-warung": UtensilsCrossed,
  tailor: Scissors,
  "local-clinic": Stethoscope,
  laundry: WashingMachine,
  pharmacy: Pill,
  mosque: Mosque,
  "worship-other": Church,
  "police-post": ShieldCheck,
  hospital: Hospital,
  puskesmas: HeartPulse,
  "bank-atm": Banknote,
  minimarket: Store,
  "transit-stop": TrainFront,
  "public-service": Building2,
  "water-gas": Droplets,
  "print-copy": Printer,
};

export function categoryStyle(category: Category): CategoryStyle {
  return {
    icon: CATEGORY_ICONS[category.slug] ?? MapPin,
    label: category.label,
    ...CATEGORY_COLORS[category.color],
  };
}

/** The category's icon in its deep tone on a soft-tinted tile. */
export function CategoryTile({
  category,
  className,
}: {
  category: Category;
  className?: string;
}) {
  const style = categoryStyle(category);

  return (
    <span
      aria-hidden="true"
      className={cn(
        "inline-grid size-8 shrink-0 place-items-center rounded-lg",
        style.tint,
        className,
      )}
    >
      <style.icon className="size-4" />
    </span>
  );
}

/** The category's icon and label as a soft-tinted pill, e.g. on a place's details. */
export function CategoryBadge({ category }: { category: Category }) {
  const style = categoryStyle(category);

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium",
        style.tint,
      )}
    >
      <style.icon aria-hidden="true" className="size-3.5" />
      {style.label}
    </span>
  );
}
