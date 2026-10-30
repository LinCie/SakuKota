import { useNavigate } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useState } from "react";

import { useDeleteCategory } from "~/features/places/queries";
import type { CategoryDetail } from "~/features/places/types";
import { Button } from "~/shared/components/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { PageHeader } from "~/shared/components/page-header";
import { SearchField } from "~/shared/components/search-field";
import { formatNumber } from "~/shared/format";
import { useDisclosure, useTableQuery } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { CategoriesTable } from "../components/categories-table";
import { CategoryFormDrawer } from "../components/category-form-drawer";

function categorySearchText(category: CategoryDetail) {
  return [category.name, category.slug].join(" ");
}

/** The category list: search, a table with row actions, and the add, edit, and delete flows. */
export function CategoriesPage({
  categories,
}: {
  categories: CategoryDetail[];
}) {
  const navigate = useNavigate();
  const search = useTableQuery(categories, categorySearchText);
  const drawer = useDisclosure();
  // Null while adding; the category being edited otherwise.
  const [editing, setEditing] = useState<CategoryDetail | null>(null);

  const deleteCategory = useDeleteCategory();
  // The category awaiting delete confirmation.
  const [deleting, setDeleting] = useState<CategoryDetail | null>(null);

  function closeDelete() {
    setDeleting(null);
    deleteCategory.reset();
  }

  function confirmDelete() {
    if (!deleting) return;
    const category = deleting;
    deleteCategory.mutate(category.id, {
      onSuccess: () => {
        toastManager.add({ title: `Kategori dihapus: ${category.name}` });
        setDeleting(null);
      },
    });
  }

  const deleteError = !deleteCategory.error
    ? null
    : hasErrorCode(deleteCategory.error, "category_in_use")
      ? "Kategori ini masih dipakai tempat. Pindahkan atau hapus tempatnya dulu."
      : writeErrorMessage(deleteCategory.error);

  function openCreate() {
    setEditing(null);
    drawer.setOpen(true);
  }

  function openEdit(category: CategoryDetail) {
    setEditing(category);
    drawer.setOpen(true);
  }

  return (
    <>
      <PageHeader
        title="Kategori"
        description="Jenis tempat dan field tambahan yang diisi untuk tiap tempat."
        actions={
          <Button variant="primary" onClick={openCreate}>
            <Plus aria-hidden="true" className="size-4" strokeWidth={2.5} />
            Tambah kategori
          </Button>
        }
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
        onEdit={openEdit}
        onDelete={setDeleting}
      />

      <CategoryFormDrawer
        key={editing?.id ?? "new"}
        open={drawer.open}
        onOpenChange={drawer.setOpen}
        category={editing ?? undefined}
      />

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) closeDelete();
        }}
        title="Hapus kategori?"
        description={`${deleting?.name ?? "Kategori ini"} akan dihapus beserta field-nya. Kategori yang masih dipakai tempat tidak bisa dihapus.`}
        confirmLabel="Hapus kategori"
        pending={deleteCategory.isPending}
        error={deleteError}
        onConfirm={confirmDelete}
      />
    </>
  );
}
