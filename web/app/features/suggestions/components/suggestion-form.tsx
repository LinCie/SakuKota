import { useForm } from "@tanstack/react-form";

import { PlaceForm } from "~/features/places/components/place-form";
import type { CategoryDetail, Place } from "~/features/places/types";
import type { PlaceValues } from "~/features/places/values";
import type { SubDistrict } from "~/features/subdistricts/types";
import { TextAreaField } from "~/shared/components/field";
import { FormAlert } from "~/shared/components/form-alert";
import { FormSection } from "~/shared/components/form-section";
import { SubmitPlate } from "~/shared/components/submit-plate";
import type { AppFormApi } from "~/shared/form-api";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { useCreateSuggestion } from "../queries";
import { emptySuggestionValues, toSuggestionRequest } from "../mappers";
import { suggestionSchema } from "../schemas";

const FORM_ID = "suggestion-form";

type SuggestionFormProps = {
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  /** Every place, so the map shows what the chosen kecamatan already has. */
  places: Place[];
};

/** The signed-in user's suggestion of a place: the place's fields and a note. */
export function SuggestionForm({
  categories,
  subDistricts,
  places,
}: SuggestionFormProps) {
  const createSuggestion = useCreateSuggestion();

  const fieldsOf = (categoryId: number) =>
    categories.find((c) => c.id === categoryId)?.fields ?? [];

  const form = useForm({
    defaultValues: emptySuggestionValues(subDistricts),
    validators: { onSubmit: suggestionSchema },
    onSubmit: ({ value, formApi }) => {
      const body = toSuggestionRequest(
        value,
        fieldsOf(Number(value.categoryId)),
      );
      createSuggestion.mutate(body, {
        onSuccess: () => {
          toastManager.add({
            title: `Usulan terkirim: ${body.name}`,
            description: "Admin akan meninjaunya sebelum tampil di peta.",
          });
          formApi.reset();
        },
      });
    },
  });

  return (
    <div className="flex flex-col gap-6">
      <PlaceForm
        // PlaceForm only touches the place's fields, which these values extend.
        form={form as unknown as AppFormApi<PlaceValues>}
        formId={FORM_ID}
        categories={categories}
        subDistricts={subDistricts}
        places={places}
      >
        <FormSection title="Untuk peninjau">
          <form.Field name="note">
            {(field) => (
              <TextAreaField
                field={field}
                label="Catatan"
                optional
                hint="Hal yang perlu diketahui admin, misalnya patokan lokasi."
              />
            )}
          </form.Field>
        </FormSection>
      </PlaceForm>

      {/* Next to the submit button, so the outcome shows where the form ends. */}
      {createSuggestion.error ? (
        <FormAlert>
          {hasErrorCode(createSuggestion.error, "TOO_MANY_PENDING_SUGGESTIONS")
            ? "Kamu sudah punya 5 usulan yang menunggu ditinjau. Kirim lagi setelah ada yang ditinjau."
            : writeErrorMessage(createSuggestion.error)}
        </FormAlert>
      ) : null}

      <SubmitPlate pending={createSuggestion.isPending} form={FORM_ID}>
        Kirim usulan
      </SubmitPlate>
    </div>
  );
}
