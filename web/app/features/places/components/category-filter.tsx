import { ChevronDown, ListFilter } from "lucide-react";

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

type CategoryFilterProps = {
  categories: Category[];
  activeSlugs: string[];
  onToggle: (slug: string) => void;
  onShowAll: () => void;
};

/**
 * The category filter, the last segment of the header's search bar: a
 * dropdown of checkable categories. On phones the trigger is an icon with the
 * number of categories picked, its label left to screen readers.
 */
export function CategoryFilter({
  categories,
  activeSlugs,
  onToggle,
  onShowAll,
}: CategoryFilterProps) {
  const activeCategories = categories.filter((category) =>
    activeSlugs.includes(category.slug),
  );
  const allActive = activeCategories.length === 0;

  let triggerLabel = `${activeCategories.length} kategori`;
  if (allActive) triggerLabel = "Semua kategori";
  else if (activeCategories.length === 1) {
    triggerLabel = activeCategories[0].label;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "hover:bg-muted aria-expanded:bg-muted focus-visible:ring-ring/50 inline-flex h-full min-w-11 shrink-0 items-center justify-center gap-1.5 rounded-r-full px-3 text-sm font-medium transition-colors outline-none focus-visible:ring-3 sm:min-w-44 sm:justify-between sm:gap-2 sm:pr-3 sm:pl-4",
          !allActive &&
            "bg-pandan-soft text-pandan-deep hover:bg-pandan-soft aria-expanded:bg-pandan-soft",
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
        <span className="sr-only max-w-40 truncate sm:not-sr-only">
          {triggerLabel}
        </span>
        <ChevronDown
          aria-hidden="true"
          className="hidden size-4 shrink-0 opacity-60 sm:block"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-auto min-w-(--anchor-width) sm:min-w-56"
      >
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
  );
}
