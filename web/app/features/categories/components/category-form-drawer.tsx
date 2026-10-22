import { useForm, useStore } from "@tanstack/react-form";
import { useRouter } from "@tanstack/react-router";

import { createCategory, updateCategory } from "~/features/places/api";
import type { CategoryDetail } from "~/features/places/types";
import { FormAlert } from "~/shared/components/form-fields";
import { FormDrawer } from "~/shared/components/form-drawer";
import { useAction } from "~/shared/hooks/use-action";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

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
  const router = useRouter();
  const action = useAction();
  const initial: CategoryValues = category
    ? { name: category.name, slug: category.slug }
    : { name: "", slug: "" };

  const form = useForm({
    defaultValues: initial,
    validators: { onSubmit: categorySchema },
    onSubmit: async ({ value }) => {
      const body = { name: value.name.trim(), slug: value.slug.trim() };

      const saved = await action.run(async () => {
        const result = category
          ? await updateCategory(category.id, body)
          : await createCategory(body);
        if (!result.ok) {
          return result.error.code === "category_exists"
            ? "Kategori dengan nama atau slug ini sudah ada."
            : writeErrorMessage(result.status, result.error);
        }
        await router.invalidate();
        return null;
      });
      if (!saved) return;

      toastManager.add({
        title: category
          ? "Perubahan kategori disimpan"
          : `Kategori ditambahkan: ${body.name}`,
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
      title={category ? "Ubah kategori" : "Tambah kategori"}
      description="Nama kategori dan slugnya."
      dirty={dirty}
      formId="category-form"
      pending={action.pending}
      submitLabel={category ? "Simpan perubahan" : "Tambah kategori"}
    >
      {action.error ? <FormAlert>{action.error}</FormAlert> : null}
      <CategoryForm form={form} formId="category-form" followSlug={!category} />
    </FormDrawer>
  );
}
