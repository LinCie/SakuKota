import { Menu } from "@base-ui/react/menu";
import { Check, ChevronDown } from "lucide-react";

import { cn } from "~/shared/cn";

import type { Category } from "../types";
import { CodePlate } from "./category-sign";

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

/** The category filter dropdown, with a count of the places it leaves showing. */
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
  const activeCategories = categories.filter((category) =>
    activeSlugs.includes(category.slug),
  );
  const allActive = activeCategories.length === 0;
  const needle = query.trim();
  const itemClass =
    "data-highlighted:bg-paint-deep flex h-10 cursor-pointer items-center gap-2.5 rounded-[3px] px-2.5 text-sm font-bold outline-none";

  return (
    <nav
      aria-label="Filter kategori"
      className="border-ink bg-paint flex flex-col gap-2 border-b-2 py-2.5 md:px-5 lg:flex-row lg:items-center lg:justify-between"
    >
      <div className="px-3 md:px-0">
        <Menu.Root>
          <Menu.Trigger
            className={cn(
              "focus-visible:outline-cat inline-flex h-10 w-full items-center gap-2 rounded-[3px] border-2 px-3 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 sm:w-auto sm:min-w-56",
              allActive
                ? "border-ink/60 bg-paint text-ink hover:border-ink border-dashed"
                : "border-ink bg-ink text-paint",
            )}
          >
            <span className="flex-1 text-left">
              {allActive
                ? "Semua kategori"
                : activeCategories.length === 1
                  ? activeCategories[0].label
                  : `${activeCategories.length} kategori`}
            </span>
            <ChevronDown
              aria-hidden="true"
              className="size-4"
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
