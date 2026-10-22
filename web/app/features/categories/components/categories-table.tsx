import { Eye, Pencil, Trash2 } from "lucide-react";

import { CategoryBadge } from "~/features/places/components/category-sign";
import type { CategoryDetail } from "~/features/places/types";
import { DataTable, type Column } from "~/shared/components/data-table";
import { RowActionsMenu } from "~/shared/components/row-actions-menu";
import { formatNumber } from "~/shared/format";

type CategoriesTableProps = {
  categories: CategoryDetail[];
  onView: (category: CategoryDetail) => void;
  onEdit: (category: CategoryDetail) => void;
  onDelete: (category: CategoryDetail) => void;
};

/** All categories, with a row menu for view, edit, and delete. */
export function CategoriesTable({
  categories,
  onView,
  onEdit,
  onDelete,
}: CategoriesTableProps) {
  const columns: Column<CategoryDetail>[] = [
    {
      key: "name",
      header: "Nama",
      pinned: true,
      className: "min-w-48",
      cell: (category) => (
        <div className="flex min-w-0 flex-col items-start gap-1">
          <CategoryBadge category={category} />
          <span className="text-ink-soft text-xs">{category.slug}</span>
        </div>
      ),
    },
    {
      key: "fields",
      header: "Jumlah field",
      className: "text-right",
      cell: (category) => (
        <span className="tabular-nums">
          {formatNumber(category.fields.length)}
        </span>
      ),
    },
    {
      key: "actions",
      header: "Aksi",
      className: "w-12 text-right",
      cell: (category) => (
        <RowActionsMenu
          label={`Aksi untuk kategori ${category.name}`}
          actions={[
            { label: "Lihat", icon: Eye, onSelect: () => onView(category) },
            { label: "Ubah", icon: Pencil, onSelect: () => onEdit(category) },
            {
              label: "Hapus",
              icon: Trash2,
              destructive: true,
              onSelect: () => onDelete(category),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      rows={categories}
      getRowKey={(category) => category.id}
      empty="Belum ada kategori yang cocok."
    />
  );
}
