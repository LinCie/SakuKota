import type { Category } from "../types";

export type CategoryStyle = {
  code: string;
  label: string;
  /** Solid painted plate: active filter, map sign, badge. */
  solid: string;
  /** Outlined plate: inactive filter. */
  outline: string;
};

// Keyed by backend category slug (cmd/seed/seed.sql). Class strings are
// complete literals so Tailwind can see them, including inside the map's
// marker HTML.
const CATEGORY_STYLES: Record<string, CategoryStyle> = {
  "affordable-food-warung": {
    code: "W",
    label: "Warung",
    solid: "bg-chili text-paint border-chili",
    outline: "border-chili text-chili",
  },
  tailor: {
    code: "J",
    label: "Penjahit",
    solid: "bg-plum text-paint border-plum",
    outline: "border-plum text-plum",
  },
  "local-clinic": {
    code: "K",
    label: "Klinik",
    solid: "bg-cat text-paint border-cat",
    outline: "border-cat text-cat",
  },
  laundry: {
    code: "L",
    label: "Laundry",
    solid: "bg-sign text-ink border-ink",
    outline: "border-ink/60 text-ink",
  },
  pharmacy: {
    code: "A",
    label: "Apotek",
    solid: "bg-board text-paint border-board",
    outline: "border-board text-board",
  },
};

export function categoryStyle(category: Category): CategoryStyle {
  return (
    CATEGORY_STYLES[category.slug] ?? {
      code: category.name.charAt(0).toUpperCase(),
      label: category.name,
      solid: "bg-ink text-paint border-ink",
      outline: "border-ink text-ink",
    }
  );
}

/** Small painted code plate, e.g. the "W" on a warung sign. */
export function CodePlate({
  category,
  className = "",
}: {
  category: Category;
  className?: string;
}) {
  const style = categoryStyle(category);

  return (
    <span
      aria-hidden="true"
      className={`font-sign inline-grid size-7 shrink-0 place-items-center rounded-[3px] border-2 text-sm leading-none ${style.solid} ${className}`}
    >
      {style.code}
    </span>
  );
}

/** Code plate plus label, used for a place's category on the menu board. */
export function CategoryBadge({ category }: { category: Category }) {
  const style = categoryStyle(category);

  return (
    <span className="text-ink inline-flex items-center gap-2 text-sm font-semibold">
      <CodePlate category={category} className="size-6 text-xs" />
      {style.label}
    </span>
  );
}
