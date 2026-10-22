import { Eye, Pencil, Trash2 } from "lucide-react";

import { DataTable, type Column } from "~/shared/components/data-table";
import { MissingSlot } from "~/shared/components/missing-slot";
import { RowActionsMenu } from "~/shared/components/row-actions-menu";
import { formatRupiah } from "~/shared/format";

import type { SubDistrict } from "../types";

type SubDistrictsTableProps = {
  subDistricts: SubDistrict[];
  onView: (subDistrict: SubDistrict) => void;
  onEdit: (subDistrict: SubDistrict) => void;
  onDelete: (subDistrict: SubDistrict) => void;
};

/** All kecamatan, with a row menu for view, edit, and delete. */
export function SubDistrictsTable({
  subDistricts,
  onView,
  onEdit,
  onDelete,
}: SubDistrictsTableProps) {
  const columns: Column<SubDistrict>[] = [
    {
      key: "name",
      header: "Nama",
      pinned: true,
      className: "min-w-48",
      cell: (subDistrict) => (
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="font-semibold">Kec. {subDistrict.name}</span>
          <span className="text-ink-soft text-xs">{subDistrict.slug}</span>
        </div>
      ),
    },
    {
      key: "flood",
      header: "Riwayat banjir",
      cell: (subDistrict) =>
        subDistrict.flood_history ? (
          <span className="line-clamp-2 max-w-96 text-pretty">
            {subDistrict.flood_history}
          </span>
        ) : (
          <MissingSlot>Data tidak tersedia</MissingSlot>
        ),
    },
    {
      key: "spending",
      header: "Pengeluaran harian",
      className: "text-right",
      cell: (subDistrict) =>
        subDistrict.estimated_daily_spending !== null ? (
          <span className="font-semibold tabular-nums">
            {formatRupiah(subDistrict.estimated_daily_spending)}
          </span>
        ) : (
          <MissingSlot>Data tidak tersedia</MissingSlot>
        ),
    },
    {
      key: "actions",
      header: "Aksi",
      className: "w-12 text-right",
      cell: (subDistrict) => (
        <RowActionsMenu
          label={`Aksi untuk Kec. ${subDistrict.name}`}
          actions={[
            { label: "Lihat", icon: Eye, onSelect: () => onView(subDistrict) },
            {
              label: "Ubah",
              icon: Pencil,
              onSelect: () => onEdit(subDistrict),
            },
            {
              label: "Hapus",
              icon: Trash2,
              destructive: true,
              onSelect: () => onDelete(subDistrict),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      rows={subDistricts}
      getRowKey={(subDistrict) => subDistrict.id}
      empty="Belum ada kecamatan yang cocok."
    />
  );
}
