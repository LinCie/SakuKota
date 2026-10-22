import { useStore } from "@tanstack/react-form";

import {
  SelectField,
  TextAreaField,
  TextField,
} from "~/shared/components/form-fields";
import { CheckboxField } from "~/shared/components/checkbox-field";
import { FormSection } from "~/shared/components/form-section";
import type { AppFormApi } from "~/shared/form-api";
import { useSlugFollow } from "~/shared/hooks/use-slug-follow";

import { FIELD_TYPES } from "../field-types";
import type { CategoryFieldValues } from "../schemas";

type CategoryFieldFormProps = {
  form: AppFormApi<CategoryFieldValues>;
  formId: string;
};

/** The fields of a category's custom-field form; the drawer owns the form and its submit. */
export function CategoryFieldForm({ form, formId }: CategoryFieldFormProps) {
  const dataType = useStore(form.store, (state) => state.values.dataType);
  // The key follows the label until it is edited by hand.
  const keyFollow = useSlugFollow(true);

  return (
    <form
      id={formId}
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
      className="flex flex-col gap-8"
    >
      <FormSection title="Field">
        <form.Field name="label">
          {(field) => (
            <TextField
              field={field}
              label="Label"
              hint="Nama yang tampil di formulir dan detail tempat."
              autoComplete="off"
              onValueChange={(value) => {
                const next = keyFollow.slugFor(value);
                if (next !== null) form.setFieldValue("key", next);
              }}
            />
          )}
        </form.Field>
        <form.Field name="key">
          {(field) => (
            <TextField
              field={field}
              label="Kunci"
              hint="Nama internal untuk menyimpan nilainya."
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              onValueChange={keyFollow.markEdited}
            />
          )}
        </form.Field>
        <form.Field name="dataType">
          {(field) => (
            <SelectField
              field={field}
              label="Tipe data"
              placeholder="Pilih tipe"
              options={FIELD_TYPES.map((type) => ({
                value: type.value,
                label: type.label,
              }))}
            />
          )}
        </form.Field>
        {dataType === "select" ? (
          <form.Field name="options">
            {(field) => (
              <TextAreaField
                field={field}
                label="Pilihan"
                hint="Satu pilihan per baris."
                rows={4}
              />
            )}
          </form.Field>
        ) : null}
        <form.Field name="required">
          {(field) => (
            <CheckboxField
              id="field-required"
              label="Wajib diisi"
              hint="Tempat tanpa nilai untuk field ini tidak bisa disimpan."
              checked={field.state.value}
              onCheckedChange={(checked) => field.handleChange(checked)}
            />
          )}
        </form.Field>
      </FormSection>
    </form>
  );
}
