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

export type CategoryStyle = {
  icon: LucideIcon;
  label: string;
  /** Solid painted plate: active filter, map sign, badge. */
  solid: string;
  /** Outlined plate: inactive filter. */
  outline: string;
};

// The paints a category's `color` selects. Class strings are complete literals
// so Tailwind can see them, including inside the map's marker HTML.
const CATEGORY_COLORS: Record<
  CategoryColor,
  Pick<CategoryStyle, "solid" | "outline">
> = {
  chili: {
    solid: "bg-chili text-paint border-chili",
    outline: "border-chili text-chili",
  },
  plum: {
    solid: "bg-plum text-paint border-plum",
    outline: "border-plum text-plum",
  },
  cat: {
    solid: "bg-cat text-paint border-cat",
    outline: "border-cat text-cat",
  },
  sign: {
    solid: "bg-sign text-ink border-ink",
    outline: "border-ink/60 text-ink",
  },
  board: {
    solid: "bg-board text-paint border-board",
    outline: "border-board text-board",
  },
};

// The icon a category shows on its sign, keyed by the migrations' `slug`.
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

/** Small painted plate with the category's icon, e.g. the cutlery on a warung sign. */
export function CodePlate({
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
        "inline-grid size-7 shrink-0 place-items-center rounded-[3px] border-2",
        style.solid,
        className,
      )}
    >
      <style.icon className="size-4" strokeWidth={2.5} />
    </span>
  );
}

/** Code plate plus label, used for a place's category on the menu board. */
export function CategoryBadge({ category }: { category: Category }) {
  const style = categoryStyle(category);

  return (
    <span className="text-ink inline-flex items-center gap-2 text-sm font-semibold">
      <CodePlate category={category} className="size-6" />
      {style.label}
    </span>
  );
}
