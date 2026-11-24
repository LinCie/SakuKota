import { ChevronDown, ListFilter } from "lucide-react";
import type { ReactNode } from "react";

import { buttonVariants } from "~/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "~/shared/components/ui/dropdown-menu";
import { cn } from "~/shared/cn";

import type { Category } from "../types";
import { CategoryTile } from "./category-sign";

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
 * On phones the row carries the search field and an icon-sized filter, and
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

  let triggerLabel = `${activeCategories.length} kategori`;
  if (allActive) triggerLabel = "Semua kategori";
  else if (activeCategories.length === 1) {
    triggerLabel = activeCategories[0].label;
  }

  return (
    <nav
      aria-label="Filter kategori"
      className="bg-background flex items-center justify-between gap-2 border-b px-3 py-2 sm:gap-3 md:px-5 md:py-2.5"
    >
      <div className="min-w-0 flex-1 sm:hidden">{search}</div>
      <div className="shrink-0">
        <DropdownMenu>
          <DropdownMenuTrigger
            className={cn(
              buttonVariants({ variant: "outline" }),
              "min-w-11 gap-1.5 px-3 sm:min-w-56 sm:justify-start sm:gap-2 sm:px-4",
              !allActive &&
                "bg-pandan-soft text-pandan-deep hover:bg-pandan-soft hover:text-pandan-deep aria-expanded:bg-pandan-soft aria-expanded:text-pandan-deep border-transparent",
            )}
          >
            <ListFilter aria-hidden="true" className="size-5 sm:hidden" />
            {allActive ? null : (
              <span
                aria-hidden="true"
                className="font-semibold tabular-nums sm:hidden"
              >
                {activeCategories.length}
              </span>
            )}
            <span className="sr-only flex-1 text-left sm:not-sr-only">
              {triggerLabel}
            </span>
            <ChevronDown
              aria-hidden="true"
              className="hidden opacity-60 sm:block"
            />
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-auto min-w-(--anchor-width) sm:min-w-56">
            <DropdownMenuItem
              closeOnClick={false}
              disabled={allActive}
              onClick={onShowAll}
              className="font-medium"
            >
              Semua kategori
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            {categories.map((category) => (
              <DropdownMenuCheckboxItem
                key={category.id}
                checked={activeSlugs.includes(category.slug)}
                onCheckedChange={() => onToggle(category.slug)}
                className="pl-1.5 font-medium"
              >
                <CategoryTile category={category} />
                {category.label}
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <p
        aria-live="polite"
        className="text-muted-foreground sr-only shrink-0 text-sm sm:not-sr-only"
      >
        <span className="text-foreground font-semibold tabular-nums">
          {count}
        </span>{" "}
        tempat di Kec. {areaName}
        {needle ? <> untuk “{query}”</> : null}
      </p>
    </nav>
  );
}
