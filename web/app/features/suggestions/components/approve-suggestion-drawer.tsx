import { useForm, useSelector } from "@tanstack/react-form";

import { PlaceForm } from "~/features/places/components/place-form";
import { toPlaceRequest } from "~/features/places/place-values";
import { placeSchema } from "~/features/places/schemas";
import type { CategoryDetail, Place } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { useApproveSuggestion } from "../queries";
import { suggestionToPlaceValues } from "../suggestion-values";
import type { Suggestion } from "../types";

const FORM_ID = "approve-suggestion-form";

type ApproveSuggestionDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  suggestion: Suggestion;
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  places: Place[];
};

/**
 * Approves `suggestion` as a place, in a drawer holding the place form filled
 * from the suggestion, for the admin to check and complete first. Mount it
 * with a key per suggestion.
 */
export function ApproveSuggestionDrawer({
  open,
  onOpenChange,
  suggestion,
  categories,
  subDistricts,
  places,
}: ApproveSuggestionDrawerProps) {
  const approve = useApproveSuggestion();

  const fieldsOf = (categoryId: number) =>
    categories.find((c) => c.id === categoryId)?.fields ?? [];
  const initial = suggestionToPlaceValues(
    suggestion,
    fieldsOf(suggestion.category_id),
  );

  const form = useForm({
    defaultValues: initial,
    validators: { onSubmit: placeSchema },
    onSubmit: ({ value }) => {
      const body = toPlaceRequest(value, fieldsOf(Number(value.categoryId)));
      approve.mutate(
        { id: suggestion.id, body },
        {
          onSuccess: () => {
            toastManager.add({ title: `Usulan disetujui: ${body.name}` });
            onOpenChange(false);
          },
        },
      );
    },
  });

  const dirty = useSelector(form.store, (state) => state.isDirty);

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(initial);
        approve.reset();
      }}
      title="Setujui usulan"
      description="Periksa dan lengkapi data tempatnya. Tempat ini masuk ke peta saat usulan disetujui."
      dirty={dirty}
      formId={FORM_ID}
      pending={approve.isPending}
      submitLabel="Setujui dan tambahkan"
    >
      {approve.error ? (
        <FormAlert>
          {hasErrorCode(approve.error, "ALREADY_REVIEWED")
            ? "Usulan ini sudah ditinjau."
            : writeErrorMessage(approve.error)}
        </FormAlert>
      ) : null}
      <PlaceForm
        form={form}
        formId={FORM_ID}
        categories={categories}
        subDistricts={subDistricts}
        places={places}
      />
    </FormDrawer>
  );
}
