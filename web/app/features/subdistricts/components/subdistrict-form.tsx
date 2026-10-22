import { TextAreaField, TextField } from "~/shared/components/form-fields";
import { FormSection } from "~/shared/components/form-section";
import type { AppFormApi } from "~/shared/form-api";
import { useSlugFollow } from "~/shared/hooks/use-slug-follow";

import type { SubDistrictValues } from "../subdistrict-values";

type SubDistrictFormProps = {
  form: AppFormApi<SubDistrictValues>;
  formId: string;
  /** Whether the slug follows the name; off when editing an existing kecamatan. */
  followSlug: boolean;
};

/** The fields of the kecamatan form; the drawer owns the form and its submit. */
export function SubDistrictForm({
  form,
  formId,
  followSlug,
}: SubDistrictFormProps) {
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
      <FormSection title="Identitas">
        <form.Field name="name">
          {(field) => (
            <TextField
              field={field}
              label="Nama kecamatan"
              autoComplete="off"
              onValueChange={(value) => {
                const next = slug.slugFor(value);
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
              hint="Dipakai di alamat peta, misalnya ?kecamatan=setiabudi."
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              onValueChange={slug.markEdited}
            />
          )}
        </form.Field>
      </FormSection>

      <FormSection title="Info area">
        <form.Field name="floodHistory">
          {(field) => (
            <TextAreaField
              field={field}
              label="Riwayat banjir"
              optional
              rows={4}
            />
          )}
        </form.Field>
        <form.Field name="safetyInfo">
          {(field) => (
            <TextAreaField
              field={field}
              label="Keamanan & kriminalitas"
              optional
              rows={4}
            />
          )}
        </form.Field>
        <form.Field name="dailySpending">
          {(field) => (
            <TextField
              field={field}
              label="Perkiraan pengeluaran harian"
              optional
              prefix="Rp"
              inputMode="numeric"
              autoComplete="off"
              placeholder="75000"
              className="tabular-nums"
            />
          )}
        </form.Field>
        <form.Field name="mapImageUrl">
          {(field) => (
            <TextField
              field={field}
              label="Alamat gambar peta"
              optional
              type="url"
              inputMode="url"
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              placeholder="https://"
            />
          )}
        </form.Field>
      </FormSection>
    </form>
  );
}
