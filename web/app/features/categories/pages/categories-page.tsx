import { useNavigate } from "@tanstack/react-router";

import type { CategoryDetail } from "~/features/places/types";
import { PageHeader } from "~/shared/components/page-header";
import { SearchField } from "~/shared/components/search-field";
import { formatNumber } from "~/shared/format";
import { useTableQuery } from "~/shared/hooks";

import { CategoriesTable } from "../components/categories-table";

function categorySearchText(category: CategoryDetail) {
  return [category.name, category.slug].join(" ");
}

/** The category list: search and a table with a row action to open one. */
export function CategoriesPage({
  categories,
}: {
  categories: CategoryDetail[];
}) {
  const navigate = useNavigate();
  const search = useTableQuery(categories, categorySearchText);

  return (
    <>
      <PageHeader
        title="Kategori"
        description="Jenis tempat dan field tambahan yang diisi untuk tiap tempat."
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <SearchField
          value={search.query}
          onValueChange={search.setQuery}
          label="Cari kategori"
          placeholder="Cari nama atau slug"
        />
        <p className="text-ink-soft text-sm tabular-nums">
          {formatNumber(search.rows.length)} dari{" "}
          {formatNumber(categories.length)} kategori
        </p>
      </div>

      <CategoriesTable
        categories={search.rows}
        onView={(category) =>
          void navigate({
            to: "/admin/categories/$categoryId",
            params: { categoryId: String(category.id) },
          })
        }
      />
    </>
  );
}
