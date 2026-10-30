import { useForm, useSelector } from "@tanstack/react-form";

import {
  useCreateCategory,
  useUpdateCategory,
} from "~/features/places/queries";
import type { CategoryDetail } from "~/features/places/types";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { categorySchema, type CategoryValues } from "../schemas";
import { CategoryForm } from "./category-form";

type CategoryFormDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The category being edited; absent when adding one. */
  category?: CategoryDetail;
};

/** Adds a category, or edits `category`, in a drawer. Mount it with a key per category. */
export function CategoryFormDrawer({
  open,
  onOpenChange,
  category,
}: CategoryFormDrawerProps) {
  const createCategory = useCreateCategory();
  const updateCategory = useUpdateCategory();
  const mutation = category ? updateCategory : createCategory;
  const initial: CategoryValues = category
    ? { name: category.name, slug: category.slug }
    : { name: "", slug: "" };

  const form = useForm({
    defaultValues: initial,
    validators: { onSubmit: categorySchema },
    onSubmit: ({ value }) => {
      const body = { name: value.name.trim(), slug: value.slug.trim() };
      const onSuccess = () => {
        toastManager.add({
          title: category
            ? "Perubahan kategori disimpan"
            : `Kategori ditambahkan: ${body.name}`,
        });
        onOpenChange(false);
      };

      if (category) {
        updateCategory.mutate({ id: category.id, body }, { onSuccess });
      } else {
        createCategory.mutate(body, { onSuccess });
      }
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
      title={category ? "Ubah kategori" : "Tambah kategori"}
      description="Nama kategori dan slugnya."
      dirty={dirty}
      formId="category-form"
      pending={mutation.isPending}
      submitLabel={category ? "Simpan perubahan" : "Tambah kategori"}
    >
      {mutation.error ? (
        <FormAlert>
          {hasErrorCode(mutation.error, "category_exists")
            ? "Kategori dengan nama atau slug ini sudah ada."
            : writeErrorMessage(mutation.error)}
        </FormAlert>
      ) : null}
      <CategoryForm form={form} formId="category-form" followSlug={!category} />
    </FormDrawer>
  );
}
