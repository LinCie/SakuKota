import { useNavigate } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useState } from "react";

import { Button } from "~/shared/components/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { PageHeader } from "~/shared/components/page-header";
import { SearchField } from "~/shared/components/search-field";
import { formatNumber } from "~/shared/format";
import { useDisclosure, useTableQuery } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import type { SubDistrict } from "../types";
import { SubDistrictFormDrawer } from "../components/subdistrict-form-drawer";
import { SubDistrictsTable } from "../components/subdistricts-table";
import { useDeleteSubDistrict } from "../queries";

function subDistrictSearchText(subDistrict: SubDistrict) {
  return [subDistrict.name, subDistrict.slug].join(" ");
}

/** The kecamatan list: search, a table with row actions, and the add, edit, and delete flows. */
export function SubDistrictsPage({
  subDistricts,
}: {
  subDistricts: SubDistrict[];
}) {
  const navigate = useNavigate();
  const search = useTableQuery(subDistricts, subDistrictSearchText);
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
    const subDistrict = deleting;
    deleteSubDistrict.mutate(subDistrict.id, {
      onSuccess: () => {
        toastManager.add({ title: `Kecamatan dihapus: ${subDistrict.name}` });
        setDeleting(null);
      },
    });
  }

  const deleteError = !deleteSubDistrict.error
    ? null
    : hasErrorCode(deleteSubDistrict.error, "sub_district_in_use")
      ? "Kecamatan ini masih punya tempat. Pindahkan atau hapus tempatnya dulu."
      : writeErrorMessage(deleteSubDistrict.error);

  function openCreate() {
    setEditing(null);
    drawer.setOpen(true);
  }

  function openEdit(subDistrict: SubDistrict) {
    setEditing(subDistrict);
    drawer.setOpen(true);
  }

  return (
    <>
      <PageHeader
        title="Kecamatan"
        description="Info area untuk peta: banjir, keamanan, dan perkiraan biaya."
        actions={
          <Button variant="primary" onClick={openCreate}>
            <Plus aria-hidden="true" className="size-4" strokeWidth={2.5} />
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
        <p className="text-ink-soft text-sm tabular-nums">
          {formatNumber(search.rows.length)} dari{" "}
          {formatNumber(subDistricts.length)} kecamatan
        </p>
      </div>

      <SubDistrictsTable
        subDistricts={search.rows}
        onView={(subDistrict) =>
          void navigate({
            to: "/admin/subdistricts/$subDistrictId",
            params: { subDistrictId: String(subDistrict.id) },
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
        description={`Kec. ${deleting?.name ?? ""} akan dihapus. Kecamatan yang masih punya tempat tidak bisa dihapus.`}
        confirmLabel="Hapus kecamatan"
        pending={deleteSubDistrict.isPending}
        error={deleteError}
        onConfirm={confirmDelete}
      />
    </>
  );
}
