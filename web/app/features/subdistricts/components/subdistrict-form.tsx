import { TextAreaField, TextField } from "~/shared/components/field";
import { FormSection } from "~/shared/components/form-section";
import type { AppFormApi } from "~/shared/form-api";
import { useSlugFollow } from "~/shared/hooks";

import type { SubDistrictValues } from "../mappers";

type SubDistrictFormProps = {
  form: AppFormApi<SubDistrictValues>;
  formId: string;
  /** Fills the slug from the name until the slug is edited, as when adding. */
  followSlug: boolean;
};

/** The fields of the kecamatan form; the drawer owns the form and its submit. */
export function SubDistrictForm({
  form,
  formId,
  followSlug,
}: SubDistrictFormProps) {
  // Lives in the drawer's content, which unmounts on close, so each opening
  // follows the name afresh.
  const slug = useSlugFollow(followSlug);

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
      <FormSection title="Dasar">
        <form.Field name="name">
          {(field) => (
            <TextField
              field={field}
              label="Nama kecamatan"
              autoComplete="off"
              placeholder="Menteng"
              onValueChange={(name) => {
                const next = slug.slugFor(name);
                if (next !== null) form.setFieldValue("slug", next);
              }}
            />
          )}
        </form.Field>
        <form.Field name="slug">
          {(field) => (
            <TextField
              field={field}
              label="Slug"
              hint="Dipakai di alamat halaman peta. Terisi dari nama; boleh diubah."
              autoComplete="off"
              placeholder="menteng"
              onValueChange={slug.markEdited}
            />
          )}
        </form.Field>
      </FormSection>

      <FormSection title="Info kecamatan">
        <form.Field name="dailySpending">
          {(field) => (
            <TextField
              field={field}
              label="Biaya harian"
              optional
              prefix="Rp"
              inputMode="decimal"
              autoComplete="off"
              placeholder="75000"
              hint="Tanpa titik ribuan."
            />
          )}
        </form.Field>
        <form.Field name="floodHistory">
          {(field) => (
            <TextAreaField field={field} label="Riwayat banjir" optional />
          )}
        </form.Field>
        <form.Field name="safetyInformation">
          {(field) => (
            <TextAreaField field={field} label="Info keamanan" optional />
          )}
        </form.Field>
      </FormSection>
    </form>
  );
}
