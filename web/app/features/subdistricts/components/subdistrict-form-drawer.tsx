import { useForm, useStore } from "@tanstack/react-form";
import { useRouter } from "@tanstack/react-router";

import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { useAction } from "~/shared/hooks";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import { createSubDistrict, updateSubDistrict } from "../api";
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
  const router = useRouter();
  const action = useAction();
  const initial = subDistrict
    ? subDistrictToValues(subDistrict)
    : emptySubDistrictValues;

  const form = useForm({
    defaultValues: initial,
    validators: { onSubmit: subDistrictSchema },
    onSubmit: async ({ value }) => {
      const body = toSubDistrictRequest(value);

      const saved = await action.run(async () => {
        const result = subDistrict
          ? await updateSubDistrict(subDistrict.id, body)
          : await createSubDistrict(body);
        if (!result.ok) {
          return result.error.code === "sub_district_exists"
            ? "Kecamatan dengan nama atau slug ini sudah ada."
            : writeErrorMessage(result.status, result.error);
        }
        await router.invalidate();
        return null;
      });
      if (!saved) return;

      toastManager.add({
        title: subDistrict
          ? "Perubahan kecamatan disimpan"
          : `Kecamatan ditambahkan: ${body.name}`,
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
      title={subDistrict ? "Ubah kecamatan" : "Tambah kecamatan"}
      description="Identitas dan info area untuk peta dan perkiraan biaya."
      dirty={dirty}
      formId="subdistrict-form"
      pending={action.pending}
      submitLabel={subDistrict ? "Simpan perubahan" : "Tambah kecamatan"}
    >
      {action.error ? <FormAlert>{action.error}</FormAlert> : null}
      <SubDistrictForm
        form={form}
        formId="subdistrict-form"
        followSlug={!subDistrict}
      />
    </FormDrawer>
  );
}
