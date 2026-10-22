import { useForm, useStore } from "@tanstack/react-form";
import { useRouter } from "@tanstack/react-router";

import type { SubDistrict } from "~/features/subdistricts/types";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { useAction } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import { createPlace, updatePlace } from "../api";
import {
  emptyPlaceValues,
  placeToValues,
  toPlaceRequest,
} from "../place-values";
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
  const router = useRouter();
  const action = useAction();

  const fieldsOf = (categoryId: number) =>
    categories.find((c) => c.id === categoryId)?.fields ?? [];
  const initial = place
    ? placeToValues(place, fieldsOf(place.category.id))
    : emptyPlaceValues(subDistricts);

  const form = useForm({
    defaultValues: initial,
    validators: { onSubmit: placeSchema },
    onSubmit: async ({ value }) => {
      const body = toPlaceRequest(value, fieldsOf(Number(value.categoryId)));

      const saved = await action.run(async () => {
        const result = place
          ? await updatePlace(place.id, body)
          : await createPlace(body);
        if (!result.ok) return writeErrorMessage(result.status, result.error);
        await router.invalidate();
        return null;
      });
      if (!saved) return;

      toastManager.add({
        title: place
          ? "Perubahan tempat disimpan"
          : `Tempat ditambahkan: ${body.name}`,
      });
      onOpenChange(false);
    },
  });

  const dirty = useStore(form.store, (state) => state.isDirty);

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(initial);
        action.clearError();
      }}
      title={place ? "Ubah tempat" : "Tambah tempat"}
      description="Kecamatan, kategori, lokasi, dan detail tempat."
      dirty={dirty}
      formId="place-form"
      pending={action.pending}
      submitLabel={place ? "Simpan perubahan" : "Tambah tempat"}
    >
      {action.error ? <FormAlert>{action.error}</FormAlert> : null}
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
