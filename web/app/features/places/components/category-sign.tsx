import { cn } from "~/shared/cn";
import type { Category, CategoryColor } from "../types";

export type CategoryStyle = {
  code: string;
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

export function categoryStyle(category: Category): CategoryStyle {
  return {
    code: category.code,
    label: category.label,
    ...CATEGORY_COLORS[category.color],
  };
}

/** Small painted code plate, e.g. the "W" on a warung sign. */
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
        "font-sign inline-grid size-7 shrink-0 place-items-center rounded-[3px] border-2 text-sm leading-none",
        style.solid,
        className,
      )}
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
