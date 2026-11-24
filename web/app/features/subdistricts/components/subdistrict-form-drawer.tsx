import { useForm, useSelector } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";

import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { toastManager } from "~/shared/toast-manager";

import { subDistrictErrorMessage } from "../errors";
import {
  emptySubDistrictValues,
  subDistrictToValues,
  toSubDistrictRequest,
} from "../mappers";
import { useCreateSubDistrict, useUpdateSubDistrict } from "../queries";
import { subDistrictSchema } from "../schemas";
import type { SubDistrict } from "../types";
import { SubDistrictForm } from "./subdistrict-form";

type SubDistrictFormDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The kecamatan being edited; absent when adding one. */
  subDistrict?: SubDistrict;
};

/**
 * Adds a kecamatan, or edits `subDistrict`, in a drawer. Mount it with a key
 * per kecamatan. A new kecamatan opens on its own page, where its boundary
 * and images are added.
 */
export function SubDistrictFormDrawer({
  open,
  onOpenChange,
  subDistrict,
}: SubDistrictFormDrawerProps) {
  const navigate = useNavigate();
  const createSubDistrict = useCreateSubDistrict();
  const updateSubDistrict = useUpdateSubDistrict();
  const mutation = subDistrict ? updateSubDistrict : createSubDistrict;

  const initial = subDistrict
    ? subDistrictToValues(subDistrict)
    : emptySubDistrictValues;

  const form = useForm({
    defaultValues: initial,
    validators: { onSubmit: subDistrictSchema },
    onSubmit: ({ value }) => {
      const body = toSubDistrictRequest(value);

      if (subDistrict) {
        updateSubDistrict.mutate(
          { id: subDistrict.id, body },
          {
            onSuccess: () => {
              toastManager.add({ title: "Perubahan kecamatan disimpan" });
              onOpenChange(false);
            },
          },
        );
        return;
      }

      createSubDistrict.mutate(body, {
        onSuccess: (created) => {
          toastManager.add({ title: `Kecamatan ditambahkan: ${created.name}` });
          onOpenChange(false);
          void navigate({
            to: "/admin/subdistricts/$subDistrictId",
            params: { subDistrictId: String(created.id) },
          });
        },
      });
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
      title={subDistrict ? "Ubah kecamatan" : "Tambah kecamatan"}
      description={
        subDistrict
          ? "Nama, slug, dan info kecamatan."
          : "Nama, slug, dan info kecamatan. Batas wilayah dan foto ditambahkan sesudahnya."
      }
      dirty={dirty}
      formId="subdistrict-form"
      pending={mutation.isPending}
      submitLabel={subDistrict ? "Simpan perubahan" : "Tambah kecamatan"}
    >
      {mutation.error ? (
        <FormAlert>{subDistrictErrorMessage(mutation.error)}</FormAlert>
      ) : null}
      <SubDistrictForm
        form={form}
        formId="subdistrict-form"
        followSlug={!subDistrict}
      />
    </FormDrawer>
  );
}
