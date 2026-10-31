import { useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Pencil, Trash2 } from "lucide-react";

import { Button } from "~/shared/components/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { DetailList, type DetailItem } from "~/shared/components/detail-list";
import { orMissing } from "~/shared/components/missing-slot";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { formatRupiah } from "~/shared/format";
import { useDisclosure } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { SubDistrictFormDrawer } from "../components/subdistrict-form-drawer";
import { subDistrictKeys, useDeleteSubDistrict } from "../queries";
import type { SubDistrict } from "../types";

/** One kecamatan: its area info, with edit and delete. */
export function SubDistrictDetailPage({
  subDistrict,
}: {
  subDistrict: SubDistrict;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const drawer = useDisclosure();
  const confirm = useDisclosure();
  const deleteSubDistrict = useDeleteSubDistrict();

  function confirmDelete() {
    deleteSubDistrict.mutate(subDistrict.id, {
      onSuccess: async () => {
        toastManager.add({ title: `Kecamatan dihapus: ${subDistrict.name}` });
        await navigate({ to: "/admin/subdistricts" });
        // Removed only once this page is gone, so it doesn't refetch the kecamatan.
        queryClient.removeQueries({
          queryKey: subDistrictKeys.detail(subDistrict.id),
        });
      },
    });
  }

  const deleteError = !deleteSubDistrict.error
    ? null
    : hasErrorCode(deleteSubDistrict.error, "SUB_DISTRICT_IN_USE")
      ? "Kecamatan ini masih punya tempat. Pindahkan atau hapus tempatnya dulu."
      : writeErrorMessage(deleteSubDistrict.error);

  const items: DetailItem[] = [
    {
      label: "Perkiraan pengeluaran harian",
      value:
        subDistrict.estimated_daily_spending !== null
          ? `${formatRupiah(subDistrict.estimated_daily_spending)} per hari`
          : orMissing(null),
    },
    {
      label: "Alamat gambar peta",
      value: subDistrict.static_map_image_url ? (
        <a
          href={subDistrict.static_map_image_url}
          target="_blank"
          rel="noreferrer"
          className="text-cat underline underline-offset-2"
        >
          {subDistrict.static_map_image_url}
        </a>
      ) : (
        orMissing(null)
      ),
    },
    {
      label: "Riwayat banjir",
      value: orMissing(subDistrict.flood_history),
      wide: true,
    },
    {
      label: "Keamanan & kriminalitas",
      value: orMissing(subDistrict.safety_crime_information),
      wide: true,
    },
  ];

  return (
    <>
      <Button
        variant="ghost"
        className="self-start"
        render={<Link to="/admin/subdistricts" />}
      >
        <ArrowLeft aria-hidden="true" className="size-4" strokeWidth={2.5} />
        Daftar kecamatan
      </Button>

      <PageHeader
        title={`Kec. ${subDistrict.name}`}
        description={`Slug: ${subDistrict.slug}`}
        actions={
          <>
            <Button onClick={() => drawer.setOpen(true)}>
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

      <SectionCard title="Info area">
        <DetailList items={items} />
      </SectionCard>

      <SubDistrictFormDrawer
        key={subDistrict.id}
        open={drawer.open}
        onOpenChange={drawer.setOpen}
        subDistrict={subDistrict}
      />

      <ConfirmDialog
        open={confirm.open}
        onOpenChange={(open) => {
          confirm.setOpen(open);
          if (!open) deleteSubDistrict.reset();
        }}
        title="Hapus kecamatan?"
        description={`Kec. ${subDistrict.name} akan dihapus. Kecamatan yang masih punya tempat tidak bisa dihapus.`}
        confirmLabel="Hapus kecamatan"
        pending={deleteSubDistrict.isPending}
        error={deleteError}
        onConfirm={confirmDelete}
      />
    </>
  );
}
