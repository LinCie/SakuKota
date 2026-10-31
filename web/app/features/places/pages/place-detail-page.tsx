import { useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Pencil, Trash2 } from "lucide-react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { Button } from "~/shared/components/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { DetailList, type DetailItem } from "~/shared/components/detail-list";
import { ImageUploadCard } from "~/shared/components/image-upload-card";
import { MissingSlot, orMissing } from "~/shared/components/missing-slot";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { useDisclosure } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import { attributeValue } from "../attribute-format";
import type { CategoryDetail, Place } from "../types";
import { CategoryBadge } from "../components/category-sign";
import { PlaceFormDrawer } from "../components/place-form-drawer";
import { placeKeys, useDeletePlace, useUploadPlaceImage } from "../queries";

type PlaceDetailPageProps = {
  place: Place;
  subDistricts: SubDistrict[];
  categories: CategoryDetail[];
  places: Place[];
};

/** One place: its details, its category's custom fields, and edit and delete. */
export function PlaceDetailPage({
  place,
  subDistricts,
  categories,
  places,
}: PlaceDetailPageProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const drawer = useDisclosure();
  const confirm = useDisclosure();
  const deletePlace = useDeletePlace();
  const uploadImage = useUploadPlaceImage();

  function confirmDelete() {
    deletePlace.mutate(place.id, {
      onSuccess: async () => {
        toastManager.add({ title: `Tempat dihapus: ${place.name}` });
        await navigate({ to: "/admin/places" });
        // Removed only once this page is gone, so it doesn't refetch the place.
        queryClient.removeQueries({ queryKey: placeKeys.detail(place.id) });
      },
    });
  }

  const deleteError = deletePlace.error
    ? writeErrorMessage(deletePlace.error)
    : null;

  const category = categories.find((c) => c.id === place.category.id);
  const fields = category?.fields ?? [];
  const subDistrict = subDistricts.find(
    (sd) => sd.id === place.sub_district_id,
  );

  const coordinates = place.location?.coordinates;
  const info: DetailItem[] = [
    {
      label: "Kecamatan",
      value: orMissing(subDistrict ? `Kec. ${subDistrict.name}` : null),
    },
    { label: "Kategori", value: <CategoryBadge category={place.category} /> },
    {
      label: "Alamat",
      value: orMissing(place.address, "Alamat tidak tersedia"),
      wide: true,
    },
    {
      label: "Jam buka",
      value: orMissing(place.operating_hours, "Jam buka tidak tersedia"),
    },
    {
      label: "Kisaran harga",
      value: orMissing(place.estimated_price_range, "Harga tidak tersedia"),
    },
    {
      label: "Koordinat",
      value: coordinates ? (
        `${coordinates[1]}, ${coordinates[0]}`
      ) : (
        <MissingSlot>Lokasi tidak tersedia</MissingSlot>
      ),
    },
    {
      label: "Deskripsi singkat",
      value: orMissing(place.short_description, "Deskripsi tidak tersedia"),
      wide: true,
    },
  ];

  const attributes: DetailItem[] = fields.map((field) => ({
    label: field.label,
    value: attributeValue(field, place),
    wide: field.data_type === "text",
  }));

  return (
    <>
      <Button
        variant="ghost"
        className="self-start"
        render={<Link to="/admin/places" />}
      >
        <ArrowLeft aria-hidden="true" className="size-4" strokeWidth={2.5} />
        Daftar tempat
      </Button>

      <PageHeader
        title={place.name}
        description={[
          category?.name ?? place.category.name,
          subDistrict ? `Kec. ${subDistrict.name}` : null,
        ]
          .filter(Boolean)
          .join(" · ")}
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

      <div className="grid gap-5 lg:grid-cols-2">
        <SectionCard title="Informasi">
          <DetailList items={info} />
        </SectionCard>
        {fields.length > 0 ? (
          <SectionCard title={`Khusus ${category?.name ?? "kategori"}`}>
            <DetailList items={attributes} />
          </SectionCard>
        ) : null}
        <ImageUploadCard
          title="Foto tempat"
          imageUrl={place.image_url}
          alt={`Foto ${place.name}`}
          missing="Foto tidak tersedia"
          aspect="aspect-4/3"
          pending={uploadImage.isPending}
          error={uploadImage.error}
          onUpload={(file) =>
            uploadImage.mutate(
              { id: place.id, file },
              {
                onSuccess: () =>
                  toastManager.add({ title: "Foto tempat disimpan" }),
              },
            )
          }
        />
      </div>

      <PlaceFormDrawer
        key={place.id}
        open={drawer.open}
        onOpenChange={drawer.setOpen}
        place={place}
        categories={categories}
        subDistricts={subDistricts}
        places={places}
      />

      <ConfirmDialog
        open={confirm.open}
        onOpenChange={(open) => {
          confirm.setOpen(open);
          if (!open) deletePlace.reset();
        }}
        title="Hapus tempat?"
        description={`${place.name} akan dihapus dari peta. Tindakan ini tidak bisa dibatalkan.`}
        confirmLabel="Hapus tempat"
        pending={deletePlace.isPending}
        error={deleteError}
        onConfirm={confirmDelete}
      />
    </>
  );
}
