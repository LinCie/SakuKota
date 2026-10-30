import { useForm, useStore } from "@tanstack/react-form";

import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { useCreateSubDistrict, useUpdateSubDistrict } from "../queries";
import { subDistrictSchema } from "../schemas";
import {
  emptySubDistrictValues,
  subDistrictToValues,
  toSubDistrictRequest,
} from "../subdistrict-values";
import type { SubDistrict } from "../types";
import { SubDistrictForm } from "./subdistrict-form";

type SubDistrictFormDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The kecamatan being edited; absent when adding one. */
  subDistrict?: SubDistrict;
};

/** Adds a kecamatan, or edits `subDistrict`, in a drawer. Mount it with a key per kecamatan. */
export function SubDistrictFormDrawer({
  open,
  onOpenChange,
  subDistrict,
}: SubDistrictFormDrawerProps) {
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
      const onSuccess = () => {
        toastManager.add({
          title: subDistrict
            ? "Perubahan kecamatan disimpan"
            : `Kecamatan ditambahkan: ${body.name}`,
        });
        onOpenChange(false);
      };

      if (subDistrict) {
        updateSubDistrict.mutate({ id: subDistrict.id, body }, { onSuccess });
      } else {
        createSubDistrict.mutate(body, { onSuccess });
      }
    },
  });

  const dirty = useStore(form.store, (state) => state.isDirty);

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(initial);
        mutation.reset();
      }}
      title={subDistrict ? "Ubah kecamatan" : "Tambah kecamatan"}
      description="Identitas dan info area untuk peta dan perkiraan biaya."
      dirty={dirty}
      formId="subdistrict-form"
      pending={mutation.isPending}
      submitLabel={subDistrict ? "Simpan perubahan" : "Tambah kecamatan"}
    >
      {mutation.error ? (
        <FormAlert>
          {hasErrorCode(mutation.error, "sub_district_exists")
            ? "Kecamatan dengan nama atau slug ini sudah ada."
            : writeErrorMessage(mutation.error)}
        </FormAlert>
      ) : null}
      <SubDistrictForm
        form={form}
        formId="subdistrict-form"
        followSlug={!subDistrict}
      />
    </FormDrawer>
  );
}
