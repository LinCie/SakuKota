import { cn } from "~/shared/cn";

import type { Category } from "../types";
import { categoryStyle } from "./category-sign";

type CategoryNavProps = {
  categories: Category[];
  activeSlugs: string[];
  onToggle: (slug: string) => void;
  onShowAll: () => void;
  /** Places on the map now, within the chosen kecamatan. */
  placeCount: number;
  /** Places matching the filters, including those the zoom limit leaves off the map. */
  totalCount: number;
  areaName: string;
  /** The search text, shown in the count when it is not blank. */
  query: string;
};

/** The category filter signs, with a count of the places they leave showing. */
export function CategoryNav({
  categories,
  activeSlugs,
  onToggle,
  onShowAll,
  placeCount,
  totalCount,
  areaName,
  query,
}: CategoryNavProps) {
  const allActive = activeSlugs.length === 0;
  const needle = query.trim();

  return (
    <nav
      aria-label="Filter kategori"
      className="border-ink bg-paint flex flex-col gap-2 border-b-2 py-2.5 md:px-5 lg:flex-row lg:items-center lg:justify-between"
    >
      <div className="flex flex-wrap gap-2 px-3 md:px-0">
        <button
          type="button"
          aria-pressed={allActive}
          onClick={onShowAll}
          className={cn(
            "focus-visible:outline-cat h-10 shrink-0 rounded-[3px] border-2 px-3 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2",
            allActive
              ? "border-ink bg-ink text-paint"
              : "border-ink/60 bg-paint text-ink hover:border-ink border-dashed",
          )}
        >
          Semua
        </button>
        {categories.map((category) => {
          const style = categoryStyle(category);
          const active = activeSlugs.includes(category.slug);

          return (
            <button
              key={category.id}
              type="button"
              aria-pressed={active}
              onClick={() => onToggle(category.slug)}
              className={cn(
                "focus-visible:outline-cat inline-flex h-10 shrink-0 items-center gap-2 rounded-[3px] border-2 pr-3 pl-1 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2",
                active
                  ? style.solid
                  : [
                      "bg-paint border-dashed hover:border-solid",
                      style.outline,
                    ],
              )}
            >
              <span
                aria-hidden="true"
                className={cn(
                  "font-sign grid size-7 place-items-center rounded-xs text-sm leading-none",
                  active ? "bg-paint/20" : ["border-2", style.solid],
                )}
              >
                {style.code}
              </span>
              {style.label}
            </button>
          );
        })}
      </div>
      <p
        aria-live="polite"
        className="text-ink-soft shrink-0 px-3 text-sm md:px-0"
      >
        <span className="font-sign text-ink text-base tabular-nums">
          {placeCount}
        </span>
        {placeCount < totalCount ? (
          <>
            {" "}
            dari{" "}
            <span className="font-sign text-ink text-base tabular-nums">
              {totalCount}
            </span>
          </>
        ) : null}{" "}
        tempat di Kec. {areaName}
        {needle ? <> untuk “{query}”</> : null}
      </p>
    </nav>
  );
}
