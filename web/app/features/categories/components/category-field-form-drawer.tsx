import { useForm, useStore } from "@tanstack/react-form";

import { useCreateCategoryField } from "~/features/places/queries";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import {
  emptyCategoryFieldValues,
  toCategoryFieldRequest,
} from "../category-field-values";
import { categoryFieldSchema } from "../schemas";
import { CategoryFieldForm } from "./category-field-form";

type CategoryFieldFormDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categoryId: number;
};

/** Adds a custom field to a category, in a drawer. */
export function CategoryFieldFormDrawer({
  open,
  onOpenChange,
  categoryId,
}: CategoryFieldFormDrawerProps) {
  const mutation = useCreateCategoryField();

  const form = useForm({
    defaultValues: emptyCategoryFieldValues,
    validators: { onSubmit: categoryFieldSchema },
    onSubmit: ({ value }) => {
      const body = toCategoryFieldRequest(value);
      mutation.mutate(
        { categoryId, body },
        {
          onSuccess: () => {
            toastManager.add({ title: `Field ditambahkan: ${body.label}` });
            onOpenChange(false);
          },
        },
      );
    },
  });

  const dirty = useStore(form.store, (state) => state.isDirty);

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(emptyCategoryFieldValues);
        mutation.reset();
      }}
      title="Tambah field"
      description="Field tambahan yang diisi untuk setiap tempat di kategori ini."
      dirty={dirty}
      formId="category-field-form"
      pending={mutation.isPending}
      submitLabel="Tambah field"
    >
      {mutation.error ? (
        <FormAlert>
          {hasErrorCode(mutation.error, "category_field_exists")
            ? "Kunci ini sudah dipakai field lain di kategori ini."
            : writeErrorMessage(mutation.error)}
        </FormAlert>
      ) : null}
      <CategoryFieldForm form={form} formId="category-field-form" />
    </FormDrawer>
  );
}
