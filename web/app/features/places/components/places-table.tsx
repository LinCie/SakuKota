import { Eye, Pencil, Trash2 } from "lucide-react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { DataTable, type Column } from "~/shared/components/data-table";
import { MissingSlot } from "~/shared/components/missing-slot";
import { RowActionsMenu } from "~/shared/components/row-actions-menu";

import type { Place } from "../types";
import { CategoryBadge } from "./category-sign";

type PlacesTableProps = {
  places: Place[];
  subDistricts: SubDistrict[];
  onView: (place: Place) => void;
  onEdit: (place: Place) => void;
  onDelete: (place: Place) => void;
};

/** All places, with a row menu for view, edit, and delete. */
export function PlacesTable({
  places,
  subDistricts,
  onView,
  onEdit,
  onDelete,
}: PlacesTableProps) {
  const subDistrictName = new Map(subDistricts.map((sd) => [sd.id, sd.name]));

  const columns: Column<Place>[] = [
    {
      key: "name",
      header: "Nama",
      pinned: true,
      className: "min-w-56",
      cell: (place) => (
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="font-semibold">{place.name}</span>
          <span className="text-muted-foreground max-w-80 truncate text-xs">
            {place.address ?? "Alamat tidak tersedia"}
          </span>
        </div>
      ),
    },
    {
      key: "category",
      header: "Kategori",
      cell: (place) => <CategoryBadge category={place.category} />,
    },
    {
      key: "sub-district",
      header: "Kecamatan",
      cell: (place) => {
        const name = subDistrictName.get(place.sub_district_id);
        return name ? (
          `Kec. ${name}`
        ) : (
          <MissingSlot>Tidak diketahui</MissingSlot>
        );
      },
    },
    {
      key: "price",
      header: "Kisaran harga",
      cell: (place) =>
        place.estimated_price_range ?? (
          <MissingSlot>Harga tidak tersedia</MissingSlot>
        ),
    },
    {
      key: "actions",
      header: "Aksi",
      className: "w-12 text-right",
      cell: (place) => (
        <RowActionsMenu
          label={`Aksi untuk ${place.name}`}
          actions={[
            { label: "Lihat", icon: Eye, onSelect: () => onView(place) },
            { label: "Ubah", icon: Pencil, onSelect: () => onEdit(place) },
            {
              label: "Hapus",
              icon: Trash2,
              destructive: true,
              onSelect: () => onDelete(place),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      rows={places}
      getRowKey={(place) => place.id}
      empty="Belum ada tempat yang cocok."
    />
  );
}
