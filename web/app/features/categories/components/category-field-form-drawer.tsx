import { useForm, useStore } from "@tanstack/react-form";
import { useRouter } from "@tanstack/react-router";

import { createCategoryField } from "~/features/places/api";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { useAction } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

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
  const router = useRouter();
  const action = useAction();

  const form = useForm({
    defaultValues: emptyCategoryFieldValues,
    validators: { onSubmit: categoryFieldSchema },
    onSubmit: async ({ value }) => {
      const body = toCategoryFieldRequest(value);

      const saved = await action.run(async () => {
        const result = await createCategoryField(categoryId, body);
        if (!result.ok) {
          return result.error.code === "category_field_exists"
            ? "Kunci ini sudah dipakai field lain di kategori ini."
            : writeErrorMessage(result.status, result.error);
        }
        await router.invalidate();
        return null;
      });
      if (!saved) return;

      toastManager.add({ title: `Field ditambahkan: ${body.label}` });
      onOpenChange(false);
    },
  });

  const dirty = useStore(form.store, (state) => state.isDirty);

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(emptyCategoryFieldValues);
        action.clearError();
      }}
      title="Tambah field"
      description="Field tambahan yang diisi untuk setiap tempat di kategori ini."
      dirty={dirty}
      formId="category-field-form"
      pending={action.pending}
      submitLabel="Tambah field"
    >
      {action.error ? <FormAlert>{action.error}</FormAlert> : null}
      <CategoryFieldForm form={form} formId="category-field-form" />
    </FormDrawer>
  );
}
