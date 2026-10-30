import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Pencil, Plus, Trash2 } from "lucide-react";

import { deleteCategory } from "~/features/places/api";
import type { CategoryDetail, CategoryField } from "~/features/places/types";
import { Button } from "~/shared/components/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { DataTable, type Column } from "~/shared/components/data-table";
import { EmptyState } from "~/shared/components/empty-state";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { useDeletion, useDisclosure } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

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
  const editDrawer = useDisclosure();
  const fieldDrawer = useDisclosure();

  const deletion = useDeletion<CategoryDetail>(
    async (target) => {
      const result = await deleteCategory(target.id);
      if (result.ok) return null;
      return result.error.code === "category_in_use"
        ? "Kategori ini masih dipakai tempat. Pindahkan atau hapus tempatnya dulu."
        : writeErrorMessage(result.status, result.error);
    },
    (target) => {
      toastManager.add({ title: `Kategori dihapus: ${target.name}` });
      void navigate({ to: "/admin/categories" });
    },
  );

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
              onClick={() => deletion.request(category)}
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
        open={deletion.target !== null}
        onOpenChange={(open) => {
          if (!open) deletion.close();
        }}
        title="Hapus kategori?"
        description={`${category.name} akan dihapus beserta field-nya. Kategori yang masih dipakai tempat tidak bisa dihapus.`}
        confirmLabel="Hapus kategori"
        pending={deletion.pending}
        error={deletion.error}
        onConfirm={() => void deletion.confirm()}
      />
    </>
  );
}
