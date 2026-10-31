import { Eye } from "lucide-react";

import { CategoryBadge } from "~/features/places/components/category-sign";
import type { CategoryDetail } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { DataTable, type Column } from "~/shared/components/data-table";
import { MissingSlot } from "~/shared/components/missing-slot";
import { RowActionsMenu } from "~/shared/components/row-actions-menu";
import { formatLocalDate } from "~/shared/format";

import type { Suggestion } from "../types";
import { SuggestionStatusBadge } from "./suggestion-status-badge";

type SuggestionsTableProps = {
  suggestions: Suggestion[];
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  onView: (suggestion: Suggestion) => void;
};

/** Suggestions, with a row action to open one for review. */
export function SuggestionsTable({
  suggestions,
  categories,
  subDistricts,
  onView,
}: SuggestionsTableProps) {
  const category = new Map(categories.map((c) => [c.id, c]));
  const subDistrictName = new Map(subDistricts.map((sd) => [sd.id, sd.name]));

  const columns: Column<Suggestion>[] = [
    {
      key: "name",
      header: "Nama",
      pinned: true,
      className: "min-w-56",
      cell: (suggestion) => (
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="font-semibold">{suggestion.name}</span>
          <span className="text-ink-soft max-w-80 truncate text-xs">
            {suggestion.address ?? "Alamat tidak tersedia"}
          </span>
        </div>
      ),
    },
    {
      key: "category",
      header: "Kategori",
      cell: (suggestion) => {
        const c = category.get(suggestion.category_id);
        return c ? (
          <CategoryBadge category={c} />
        ) : (
          <MissingSlot>Tidak diketahui</MissingSlot>
        );
      },
    },
    {
      key: "sub-district",
      header: "Kecamatan",
      cell: (suggestion) => {
        const name = subDistrictName.get(suggestion.sub_district_id);
        return name ? (
          `Kec. ${name}`
        ) : (
          <MissingSlot>Tidak diketahui</MissingSlot>
        );
      },
    },
    {
      key: "author",
      header: "Pengusul",
      cell: (suggestion) => suggestion.user.name,
    },
    {
      key: "created",
      header: "Dikirim",
      cell: (suggestion) => (
        <time dateTime={suggestion.created_at} className="tabular-nums">
          {formatLocalDate(new Date(suggestion.created_at))}
        </time>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (suggestion) => (
        <SuggestionStatusBadge status={suggestion.status} />
      ),
    },
    {
      key: "actions",
      header: "Aksi",
      className: "w-12 text-right",
      cell: (suggestion) => (
        <RowActionsMenu
          label={`Aksi untuk ${suggestion.name}`}
          actions={[
            { label: "Lihat", icon: Eye, onSelect: () => onView(suggestion) },
          ]}
        />
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      rows={suggestions}
      getRowKey={(suggestion) => suggestion.id}
      empty="Belum ada usulan yang cocok."
    />
  );
}
