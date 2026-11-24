import { useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Pencil, Trash2 } from "lucide-react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { Button, buttonVariants } from "~/shared/components/ui/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { DetailList, type DetailItem } from "~/shared/components/detail-list";
import { MediaCard } from "~/shared/components/media-card";
import { MissingSlot, orMissing } from "~/shared/components/missing-slot";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { useDisclosure } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import { attributeValue } from "../components/attribute-value";
import type { CategoryDetail, Place } from "../types";
import { CategoryBadge } from "../components/category-sign";
import { PlaceFormDrawer } from "../components/place-form-drawer";
import { hoursValue, priceValue } from "../components/schedule-values";
import {
  placeKeys,
  useAddPlaceMedia,
  useDeletePlace,
  useDeletePlaceMedia,
} from "../queries";

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
  const addMedia = useAddPlaceMedia();
  const deleteMedia = useDeletePlaceMedia();

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
    { label: "Jam buka", value: hoursValue(place.operating_hours) },
    ...(place.category.has_price
      ? [
          {
            label: "Kisaran harga",
            value: priceValue(place.estimated_price_range, place.category),
          },
        ]
      : []),
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
      <PageHeader
        back={
          <Link
            to="/admin/places"
            className={buttonVariants({
              variant: "ghost",
              size: "sm",
              className: "text-muted-foreground -ml-3 self-start",
            })}
          >
            <ArrowLeft aria-hidden="true" data-icon="inline-start" />
            Daftar tempat
          </Link>
        }
        title={place.name}
        description={[
          category?.name ?? place.category.name,
          subDistrict ? `Kec. ${subDistrict.name}` : null,
        ]
          .filter(Boolean)
          .join(" · ")}
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
        {fields.length > 0 ? (
          <SectionCard title={`Khusus ${category?.name ?? "kategori"}`}>
            <DetailList items={attributes} />
          </SectionCard>
        ) : null}
        <MediaCard
          title="Foto tempat"
          medias={place.medias}
          alt={`Foto ${place.name}`}
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
              { id: place.id, file },
              {
                onSuccess: () =>
                  toastManager.add({ title: "Foto tempat disimpan" }),
              },
            );
          }}
          onDelete={(media) => {
            addMedia.reset();
            deleteMedia.mutate(
              { id: place.id, mediaId: media.id },
              {
                onSuccess: () =>
                  toastManager.add({ title: "Foto tempat dihapus" }),
              },
            );
          }}
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
