import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Pencil, Trash2 } from "lucide-react";

import { Button } from "~/shared/components/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { DetailList, type DetailItem } from "~/shared/components/detail-list";
import { orMissing } from "~/shared/components/missing-slot";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { formatRupiah } from "~/shared/format";
import { useDeletion, useDisclosure } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import { deleteSubDistrict } from "../api";
import { SubDistrictFormDrawer } from "../components/subdistrict-form-drawer";
import type { SubDistrict } from "../types";

/** One kecamatan: its area info, with edit and delete. */
export function SubDistrictDetailPage({
  subDistrict,
}: {
  subDistrict: SubDistrict;
}) {
  const navigate = useNavigate();
  const drawer = useDisclosure();

  const deletion = useDeletion<SubDistrict>(
    async (target) => {
      const result = await deleteSubDistrict(target.id);
      if (result.ok) return null;
      return result.error.code === "sub_district_in_use"
        ? "Kecamatan ini masih punya tempat. Pindahkan atau hapus tempatnya dulu."
        : writeErrorMessage(result.status, result.error);
    },
    (target) => {
      toastManager.add({ title: `Kecamatan dihapus: ${target.name}` });
      void navigate({ to: "/admin/subdistricts" });
    },
  );

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
              onClick={() => deletion.request(subDistrict)}
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
        open={deletion.target !== null}
        onOpenChange={(open) => {
          if (!open) deletion.close();
        }}
        title="Hapus kecamatan?"
        description={`Kec. ${subDistrict.name} akan dihapus. Kecamatan yang masih punya tempat tidak bisa dihapus.`}
        confirmLabel="Hapus kecamatan"
        pending={deletion.pending}
        error={deletion.error}
        onConfirm={() => void deletion.confirm()}
      />
    </>
  );
}
