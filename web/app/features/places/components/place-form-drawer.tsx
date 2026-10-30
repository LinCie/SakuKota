import { useForm, useSelector } from "@tanstack/react-form";

import type { SubDistrict } from "~/features/subdistricts/types";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import {
  emptyPlaceValues,
  placeToValues,
  toPlaceRequest,
} from "../place-values";
import { useCreatePlace, useUpdatePlace } from "../queries";
import { placeSchema } from "../schemas";
import type { CategoryDetail, Place } from "../types";
import { PlaceForm } from "./place-form";

type PlaceFormDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The place being edited; absent when adding one. */
  place?: Place;
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  places: Place[];
};

/** Adds a place, or edits `place`, in a drawer. Mount it with a key per place. */
export function PlaceFormDrawer({
  open,
  onOpenChange,
  place,
  categories,
  subDistricts,
  places,
}: PlaceFormDrawerProps) {
  const createPlace = useCreatePlace();
  const updatePlace = useUpdatePlace();
  const mutation = place ? updatePlace : createPlace;

  const fieldsOf = (categoryId: number) =>
    categories.find((c) => c.id === categoryId)?.fields ?? [];
  const initial = place
    ? placeToValues(place, fieldsOf(place.category.id))
    : emptyPlaceValues(subDistricts);

  const form = useForm({
    defaultValues: initial,
    validators: { onSubmit: placeSchema },
    onSubmit: ({ value }) => {
      const body = toPlaceRequest(value, fieldsOf(Number(value.categoryId)));
      const onSuccess = () => {
        toastManager.add({
          title: place
            ? "Perubahan tempat disimpan"
            : `Tempat ditambahkan: ${body.name}`,
        });
        onOpenChange(false);
      };

      if (place) updatePlace.mutate({ id: place.id, body }, { onSuccess });
      else createPlace.mutate(body, { onSuccess });
    },
  });

  const dirty = useSelector(form.store, (state) => state.isDirty);

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(initial);
        mutation.reset();
      }}
      title={place ? "Ubah tempat" : "Tambah tempat"}
      description="Kecamatan, kategori, lokasi, dan detail tempat."
      dirty={dirty}
      formId="place-form"
      pending={mutation.isPending}
      submitLabel={place ? "Simpan perubahan" : "Tambah tempat"}
    >
      {mutation.error ? (
        <FormAlert>{writeErrorMessage(mutation.error)}</FormAlert>
      ) : null}
      <PlaceForm
        form={form}
        formId="place-form"
        categories={categories}
        subDistricts={subDistricts}
        places={places}
      />
    </FormDrawer>
  );
}
