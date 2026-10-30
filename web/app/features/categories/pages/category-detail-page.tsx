import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";

import type { CategoryDetail, CategoryField } from "~/features/places/types";
import { Button } from "~/shared/components/button";
import { DataTable, type Column } from "~/shared/components/data-table";
import { EmptyState } from "~/shared/components/empty-state";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";

import { fieldTypeLabel } from "../field-types";

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

/** One category: its details and custom fields. */
export function CategoryDetailPage({ category }: { category: CategoryDetail }) {
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
      />

      <SectionCard title="Field tambahan">
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
    </>
  );
}
