import { useNavigate } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useState } from "react";

import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { PageHeader } from "~/shared/components/page-header";
import { SearchField } from "~/shared/components/search-field";
import { Button } from "~/shared/components/ui/button";
import { formatNumber } from "~/shared/format";
import { useDisclosure, useTableQuery } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";

import { SubDistrictFormDrawer } from "../components/subdistrict-form-drawer";
import { SubDistrictsTable } from "../components/subdistricts-table";
import { subDistrictErrorMessage } from "../errors";
import { useDeleteSubDistrict } from "../queries";
import type { SubDistrict } from "../types";

function searchText(sd: SubDistrict): string {
  return `${sd.name} ${sd.slug}`;
}

/** The kecamatan list: search, a table with row actions, and the add, edit, and delete flows. */
export function SubDistrictsPage({
  subDistricts,
}: {
  subDistricts: SubDistrict[];
}) {
  const navigate = useNavigate();
  const search = useTableQuery(subDistricts, searchText);
  const drawer = useDisclosure();
  // Null while adding; the kecamatan being edited otherwise.
  const [editing, setEditing] = useState<SubDistrict | null>(null);

  const deleteSubDistrict = useDeleteSubDistrict();
  // The kecamatan awaiting delete confirmation.
  const [deleting, setDeleting] = useState<SubDistrict | null>(null);

  function closeDelete() {
    setDeleting(null);
    deleteSubDistrict.reset();
  }

  function confirmDelete() {
    if (!deleting) return;
    const sd = deleting;
    deleteSubDistrict.mutate(sd.id, {
      onSuccess: () => {
        toastManager.add({ title: `Kecamatan dihapus: ${sd.name}` });
        setDeleting(null);
      },
    });
  }

  function openCreate() {
    setEditing(null);
    drawer.setOpen(true);
  }

  function openEdit(sd: SubDistrict) {
    setEditing(sd);
    drawer.setOpen(true);
  }

  return (
    <>
      <PageHeader
        title="Kecamatan"
        description="Kecamatan di peta, dengan batas wilayah, foto, dan infonya."
        actions={
          <Button size="sm" onClick={openCreate}>
            <Plus aria-hidden="true" data-icon="inline-start" />
            Tambah kecamatan
          </Button>
        }
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <SearchField
          value={search.query}
          onValueChange={search.setQuery}
          label="Cari kecamatan"
          placeholder="Cari nama atau slug"
        />
        <p className="text-muted-foreground text-sm tabular-nums">
          {formatNumber(search.rows.length)} dari{" "}
          {formatNumber(subDistricts.length)} kecamatan
        </p>
      </div>

      <SubDistrictsTable
        subDistricts={search.rows}
        onView={(sd) =>
          void navigate({
            to: "/admin/subdistricts/$subDistrictId",
            params: { subDistrictId: String(sd.id) },
          })
        }
        onEdit={openEdit}
        onDelete={setDeleting}
      />

      <SubDistrictFormDrawer
        key={editing?.id ?? "new"}
        open={drawer.open}
        onOpenChange={drawer.setOpen}
        subDistrict={editing ?? undefined}
      />

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) closeDelete();
        }}
        title="Hapus kecamatan?"
        description={`Kec. ${deleting?.name ?? ""} akan dihapus beserta usulan tempat di dalamnya. Tindakan ini tidak bisa dibatalkan.`}
        confirmLabel="Hapus kecamatan"
        pending={deleteSubDistrict.isPending}
        error={
          deleteSubDistrict.error
            ? subDistrictErrorMessage(deleteSubDistrict.error)
            : null
        }
        onConfirm={confirmDelete}
      />
    </>
  );
}
