import { TextField } from "~/shared/components/form-fields";
import { FormSection } from "~/shared/components/form-section";
import type { AppFormApi } from "~/shared/form-api";
import { useSlugFollow } from "~/shared/hooks/use-slug-follow";

import type { CategoryValues } from "../schemas";

type CategoryFormProps = {
  form: AppFormApi<CategoryValues>;
  formId: string;
  /** Whether the slug follows the name; off when editing an existing category. */
  followSlug: boolean;
};

/** The fields of the category form; the drawer owns the form and its submit. */
export function CategoryForm({ form, formId, followSlug }: CategoryFormProps) {
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
              label="Nama kategori"
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
              hint="Pengenal singkat kategori, misalnya laundry."
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              onValueChange={slug.markEdited}
            />
          )}
        </form.Field>
      </FormSection>
    </form>
  );
}
