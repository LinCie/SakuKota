import { useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Pencil, Trash2 } from "lucide-react";

import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { DetailList, type DetailItem } from "~/shared/components/detail-list";
import { MediaCard } from "~/shared/components/media-card";
import { orMissing } from "~/shared/components/missing-slot";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { Button, buttonVariants } from "~/shared/components/ui/button";
import { formatRupiah } from "~/shared/format";
import { useDisclosure } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";

import { BoundaryCard } from "../components/boundary-card";
import { SubDistrictFormDrawer } from "../components/subdistrict-form-drawer";
import { subDistrictErrorMessage } from "../errors";
import {
  subDistrictKeys,
  useAddSubDistrictMedia,
  useDeleteSubDistrict,
  useDeleteSubDistrictMedia,
} from "../queries";
import type { SubDistrict } from "../types";

type SubDistrictDetailPageProps = {
  subDistrict: SubDistrict;
  /** Every kecamatan, so the boundary map can show the neighbours. */
  subDistricts: SubDistrict[];
};

/** One kecamatan: its details, boundary, and images, with edit and delete. */
export function SubDistrictDetailPage({
  subDistrict,
  subDistricts,
}: SubDistrictDetailPageProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const drawer = useDisclosure();
  const confirm = useDisclosure();
  const deleteSubDistrict = useDeleteSubDistrict();
  const addMedia = useAddSubDistrictMedia();
  const deleteMedia = useDeleteSubDistrictMedia();

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

  const spending = subDistrict.estimated_daily_spending;
  const info: DetailItem[] = [
    { label: "Nama", value: subDistrict.name },
    { label: "Slug", value: subDistrict.slug },
    {
      label: "Biaya harian",
      value: orMissing(
        spending === null ? null : formatRupiah(spending),
        "Biaya tidak tersedia",
      ),
    },
    {
      label: "Riwayat banjir",
      value: orMissing(subDistrict.flood_history, "Riwayat tidak tersedia"),
      wide: true,
    },
    {
      label: "Info keamanan",
      value: orMissing(
        subDistrict.safety_crime_information,
        "Info tidak tersedia",
      ),
      wide: true,
    },
  ];

  const others = subDistricts.flatMap((sd) =>
    sd.id !== subDistrict.id && sd.boundary ? [sd.boundary] : [],
  );

  return (
    <>
      <Link
        to="/admin/subdistricts"
        className={buttonVariants({
          variant: "ghost",
          size: "sm",
          className: "text-muted-foreground -ml-3 self-start",
        })}
      >
        <ArrowLeft aria-hidden="true" data-icon="inline-start" />
        Daftar kecamatan
      </Link>

      <PageHeader
        title={`Kec. ${subDistrict.name}`}
        description={subDistrict.slug}
        actions={
          <>
            <Button
              variant="outline"
              size="sm"
              onClick={() => drawer.setOpen(true)}
            >
              <Pencil aria-hidden="true" data-icon="inline-start" />
              Ubah
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => confirm.setOpen(true)}
            >
              <Trash2 aria-hidden="true" data-icon="inline-start" />
              Hapus
            </Button>
          </>
        }
      />

      <div className="grid gap-5 lg:grid-cols-2">
        <SectionCard title="Informasi">
          <DetailList items={info} />
        </SectionCard>
        <BoundaryCard subDistrict={subDistrict} others={others} />
        <MediaCard
          title="Foto kecamatan"
          medias={subDistrict.medias}
          alt={`Foto Kec. ${subDistrict.name}`}
          missing="Foto tidak tersedia"
          aspect="aspect-4/3"
          pending={addMedia.isPending}
          deletingId={
            deleteMedia.isPending ? deleteMedia.variables.mediaId : null
          }
          error={addMedia.error ?? deleteMedia.error}
          onAdd={(file) => {
            deleteMedia.reset();
            addMedia.mutate(
              { id: subDistrict.id, file },
              {
                onSuccess: () =>
                  toastManager.add({ title: "Foto kecamatan disimpan" }),
              },
            );
          }}
          onDelete={(media) => {
            addMedia.reset();
            deleteMedia.mutate(
              { id: subDistrict.id, mediaId: media.id },
              {
                onSuccess: () =>
                  toastManager.add({ title: "Foto kecamatan dihapus" }),
              },
            );
          }}
        />
      </div>

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
        description={`Kec. ${subDistrict.name} akan dihapus beserta usulan tempat di dalamnya. Tindakan ini tidak bisa dibatalkan.`}
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
