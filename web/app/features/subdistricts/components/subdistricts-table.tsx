import { Eye, Pencil, Trash2 } from "lucide-react";

import { DataTable, type Column } from "~/shared/components/data-table";
import { MissingSlot } from "~/shared/components/missing-slot";
import { RowActionsMenu } from "~/shared/components/row-actions-menu";
import { Badge } from "~/shared/components/ui/badge";
import { formatRupiah } from "~/shared/format";

import type { SubDistrict } from "../types";

type SubDistrictsTableProps = {
  subDistricts: SubDistrict[];
  onView: (subDistrict: SubDistrict) => void;
  onEdit: (subDistrict: SubDistrict) => void;
  onDelete: (subDistrict: SubDistrict) => void;
};

/** Every kecamatan, with a row menu for view, edit, and delete. */
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
      cell: (sd) => (
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="font-semibold">Kec. {sd.name}</span>
          <span className="text-muted-foreground text-xs">{sd.slug}</span>
        </div>
      ),
    },
    {
      key: "boundary",
      header: "Batas wilayah",
      cell: (sd) =>
        sd.boundary ? (
          <Badge className="bg-pandan-soft text-pandan-deep">Ada</Badge>
        ) : (
          <MissingSlot>Belum ada</MissingSlot>
        ),
    },
    {
      key: "spending",
      header: "Biaya harian",
      cell: (sd) =>
        sd.estimated_daily_spending === null ? (
          <MissingSlot>Biaya tidak tersedia</MissingSlot>
        ) : (
          formatRupiah(sd.estimated_daily_spending)
        ),
    },
    {
      key: "medias",
      header: "Foto",
      cell: (sd) => <span className="tabular-nums">{sd.medias.length}</span>,
    },
    {
      key: "actions",
      header: "Aksi",
      className: "w-12 text-right",
      cell: (sd) => (
        <RowActionsMenu
          label={`Aksi untuk Kec. ${sd.name}`}
          actions={[
            { label: "Lihat", icon: Eye, onSelect: () => onView(sd) },
            { label: "Ubah", icon: Pencil, onSelect: () => onEdit(sd) },
            {
              label: "Hapus",
              icon: Trash2,
              destructive: true,
              onSelect: () => onDelete(sd),
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
      getRowKey={(sd) => sd.id}
      empty="Belum ada kecamatan yang cocok."
    />
  );
}
