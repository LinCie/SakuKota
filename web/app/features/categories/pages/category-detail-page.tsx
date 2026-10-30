import { useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Pencil, Plus, Trash2 } from "lucide-react";

import { categoryKeys, useDeleteCategory } from "~/features/places/queries";
import type { CategoryDetail, CategoryField } from "~/features/places/types";
import { Button } from "~/shared/components/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { DataTable, type Column } from "~/shared/components/data-table";
import { EmptyState } from "~/shared/components/empty-state";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { useDisclosure } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { fieldTypeLabel } from "../field-types";
import { CategoryFieldFormDrawer } from "../components/category-field-form-drawer";
import { CategoryFormDrawer } from "../components/category-form-drawer";

const fieldColumns: Column<CategoryField>[] = [
  {
    key: "label",
    header: "Label",
    pinned: true,
    className: "min-w-48",
    cell: (field) => (
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="font-semibold">{field.label}</span>
        <span className="text-ink-soft text-xs">{field.key}</span>
      </div>
    ),
  },
  {
    key: "type",
    header: "Tipe",
    cell: (field) => fieldTypeLabel(field.data_type),
  },
  {
    key: "required",
    header: "Wajib",
    cell: (field) => (field.required ? "Ya" : "Tidak"),
  },
  {
    key: "options",
    header: "Pilihan",
    cell: (field) =>
      field.options ? (
        field.options.join(", ")
      ) : (
        <span className="text-ink-soft">Tidak berlaku</span>
      ),
  },
];

/** One category: its details and custom fields, with edit, delete, and add field. */
export function CategoryDetailPage({ category }: { category: CategoryDetail }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const editDrawer = useDisclosure();
  const fieldDrawer = useDisclosure();
  const confirm = useDisclosure();
  const deleteCategory = useDeleteCategory();

  function confirmDelete() {
    deleteCategory.mutate(category.id, {
      onSuccess: async () => {
        toastManager.add({ title: `Kategori dihapus: ${category.name}` });
        await navigate({ to: "/admin/categories" });
        // Removed only once this page is gone, so it doesn't refetch the category.
        queryClient.removeQueries({
          queryKey: categoryKeys.detail(category.id),
        });
      },
    });
  }

  const deleteError = !deleteCategory.error
    ? null
    : hasErrorCode(deleteCategory.error, "category_in_use")
      ? "Kategori ini masih dipakai tempat. Pindahkan atau hapus tempatnya dulu."
      : writeErrorMessage(deleteCategory.error);

  return (
    <>
      <Button
        variant="ghost"
        className="self-start"
        render={<Link to="/admin/categories" />}
      >
        <ArrowLeft aria-hidden="true" className="size-4" strokeWidth={2.5} />
        Daftar kategori
      </Button>

      <PageHeader
        title={category.name}
        description={`Slug: ${category.slug}`}
        actions={
          <>
            <Button onClick={() => editDrawer.setOpen(true)}>
              <Pencil aria-hidden="true" className="size-4" strokeWidth={2.5} />
              Ubah
            </Button>
            <Button
              className="text-chili"
              onClick={() => confirm.setOpen(true)}
            >
              <Trash2 aria-hidden="true" className="size-4" strokeWidth={2.5} />
              Hapus
            </Button>
          </>
        }
      />

      <SectionCard
        title="Field tambahan"
        actions={
          <Button onClick={() => fieldDrawer.setOpen(true)}>
            <Plus aria-hidden="true" className="size-4" strokeWidth={2.5} />
            Tambah field
          </Button>
        }
      >
        <DataTable
          columns={fieldColumns}
          rows={category.fields}
          getRowKey={(field) => field.id}
          empty={
            <EmptyState
              title="Belum ada field"
              description="Field diisi untuk setiap tempat di kategori ini, misalnya ukuran kamar atau jenis layanan."
            />
          }
        />
      </SectionCard>

      <CategoryFormDrawer
        key={category.id}
        open={editDrawer.open}
        onOpenChange={editDrawer.setOpen}
        category={category}
      />
      <CategoryFieldFormDrawer
        open={fieldDrawer.open}
        onOpenChange={fieldDrawer.setOpen}
        categoryId={category.id}
      />

      <ConfirmDialog
        open={confirm.open}
        onOpenChange={(open) => {
          confirm.setOpen(open);
          if (!open) deleteCategory.reset();
        }}
        title="Hapus kategori?"
        description={`${category.name} akan dihapus beserta field-nya. Kategori yang masih dipakai tempat tidak bisa dihapus.`}
        confirmLabel="Hapus kategori"
        pending={deleteCategory.isPending}
        error={deleteError}
        onConfirm={confirmDelete}
      />
    </>
  );
}
