import { Menu } from "@base-ui/react/menu";
import { Check, ChevronDown, ListFilter } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "~/shared/cn";

import type { Category } from "../types";
import { CodePlate } from "./category-sign";

type CategoryNavProps = {
  categories: Category[];
  activeSlugs: string[];
  onToggle: (slug: string) => void;
  onShowAll: () => void;
  /** Places matching the filters within the chosen kecamatan. */
  count: number;
  areaName: string;
  /** The search text, shown in the count when it is not blank. */
  query: string;
  /** The search field; on phones it sits here instead of in the header. */
  search: ReactNode;
};

/**
 * The category filter dropdown, with a count of the places it leaves showing.
 * On phones the strip carries the search field and an icon-sized filter, and
 * the count is left to screen readers and the list heading.
 */
export function CategoryNav({
  categories,
  activeSlugs,
  onToggle,
  onShowAll,
  count,
  areaName,
  query,
  search,
}: CategoryNavProps) {
  const activeCategories = categories.filter((category) =>
    activeSlugs.includes(category.slug),
  );
  const allActive = activeCategories.length === 0;
  const needle = query.trim();
  const itemClass =
    "data-highlighted:bg-paint-deep flex h-11 cursor-pointer items-center gap-2.5 rounded-[3px] px-2.5 text-sm font-bold outline-none md:h-10";

  return (
    <nav
      aria-label="Filter kategori"
      className="border-ink bg-paint flex items-center justify-between gap-2 border-b-2 px-3 py-2 sm:gap-3 md:px-5 md:py-2.5"
    >
      <div className="min-w-0 flex-1 sm:hidden">{search}</div>
      <div className="shrink-0">
        <Menu.Root>
          <Menu.Trigger
            className={cn(
              "focus-visible:outline-cat inline-flex h-11 min-w-11 items-center justify-center gap-1.5 rounded-[3px] border-2 px-2.5 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 sm:min-w-56 sm:justify-start sm:gap-2 sm:px-3 md:h-10",
              allActive
                ? "border-ink/60 bg-paint text-ink hover:border-ink border-dashed"
                : "border-ink bg-ink text-paint",
            )}
          >
            <ListFilter
              aria-hidden="true"
              className="size-5 sm:hidden"
              strokeWidth={2.5}
            />
            {allActive ? null : (
              <span
                aria-hidden="true"
                className="font-sign tabular-nums sm:hidden"
              >
                {activeCategories.length}
              </span>
            )}
            <span className="sr-only flex-1 text-left sm:not-sr-only">
              {allActive
                ? "Semua kategori"
                : activeCategories.length === 1
                  ? activeCategories[0].label
                  : `${activeCategories.length} kategori`}
            </span>
            <ChevronDown
              aria-hidden="true"
              className="hidden size-4 sm:block"
              strokeWidth={2.5}
            />
          </Menu.Trigger>
          <Menu.Portal>
            <Menu.Positioner sideOffset={4} align="start" className="z-50">
              <Menu.Popup className="border-ink/15 bg-paint text-ink max-h-(--available-height) min-w-(--anchor-width) overflow-y-auto rounded-[5px] border p-1 shadow-[0_8px_24px_rgb(26_26_23/0.2)] outline-none">
                <Menu.Item
                  closeOnClick={false}
                  disabled={allActive}
                  onClick={onShowAll}
                  className={cn(itemClass, "data-disabled:opacity-50")}
                >
                  Semua kategori
                </Menu.Item>
                <div aria-hidden="true" className="bg-ink/15 mx-1 my-1 h-px" />
                {categories.map((category) => (
                  <Menu.CheckboxItem
                    key={category.id}
                    checked={activeSlugs.includes(category.slug)}
                    onCheckedChange={() => onToggle(category.slug)}
                    className={itemClass}
                  >
                    <CodePlate category={category} />
                    <span className="flex-1">{category.label}</span>
                    <Menu.CheckboxItemIndicator>
                      <Check
                        aria-hidden="true"
                        className="size-4"
                        strokeWidth={3}
                      />
                    </Menu.CheckboxItemIndicator>
                  </Menu.CheckboxItem>
                ))}
              </Menu.Popup>
            </Menu.Positioner>
          </Menu.Portal>
        </Menu.Root>
      </div>
      <p
        aria-live="polite"
        className="text-ink-soft sr-only shrink-0 text-sm sm:not-sr-only"
      >
        <span className="font-sign text-ink text-base tabular-nums">
          {count}
        </span>{" "}
        tempat di Kec. {areaName}
        {needle ? <> untuk “{query}”</> : null}
      </p>
    </nav>
  );
}
