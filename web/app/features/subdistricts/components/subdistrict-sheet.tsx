import { useForm, useStore } from "@tanstack/react-form";
import { CloudRain, Landmark } from "lucide-react";
import { useRef, useState } from "react";

import { adminWriteError } from "~/shared/admin-error";
import {
  FormAlert,
  TextAreaField,
  TextField,
} from "~/shared/components/form-fields";
import { FormSheet } from "~/shared/components/form-sheet";
import { InfoSection } from "~/shared/components/menu-board";
import { createSubDistrict } from "../api";
import { parseRupiah, slugify, subDistrictSchema } from "../schemas";
import type { SubDistrict } from "../types";

const emptyValues = {
  name: "",
  slug: "",
  floodHistory: "",
  safetyInfo: "",
  dailySpending: "",
  mapImageUrl: "",
};

function orNull(text: string) {
  const trimmed = text.trim();
  return trimmed === "" ? null : trimmed;
}

type SubDistrictSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Called with the new kecamatan once it is saved, before the sheet closes. */
  onCreated: (subDistrict: SubDistrict) => Promise<void>;
};

/** The "Tambah kecamatan" form. */
export function SubDistrictSheet({
  open,
  onOpenChange,
  onCreated,
}: SubDistrictSheetProps) {
  const [formError, setFormError] = useState<string | null>(null);
  // The slug follows the name until the admin edits it.
  const slugEdited = useRef(false);

  const form = useForm({
    defaultValues: emptyValues,
    validators: { onSubmit: subDistrictSchema },
    onSubmit: async ({ value }) => {
      setFormError(null);
      try {
        const result = await createSubDistrict({
          name: value.name.trim(),
          slug: value.slug.trim(),
          flood_history: orNull(value.floodHistory),
          safety_crime_information: orNull(value.safetyInfo),
          estimated_daily_spending: parseRupiah(value.dailySpending),
          static_map_image_url: orNull(value.mapImageUrl),
        });
        if (!result.ok) {
          setFormError(
            result.error.code === "sub_district_exists"
              ? "Kecamatan dengan nama atau slug ini sudah ada."
              : adminWriteError(result.status, result.error),
          );
          return;
        }
        await onCreated(result.data);
        onOpenChange(false);
      } catch {
        setFormError("Tidak bisa terhubung ke server. Coba lagi.");
      }
    },
  });

  const pending = useStore(form.store, (state) => state.isSubmitting);
  const dirty = useStore(form.store, (state) => state.isDirty);

  return (
    <FormSheet
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset();
        slugEdited.current = false;
        setFormError(null);
      }}
      title="Tambah kecamatan"
      dirty={dirty}
      formId="subdistrict-form"
      pending={pending}
      submitLabel="Simpan kecamatan"
    >
      <form
        id="subdistrict-form"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
        className="flex flex-col gap-5"
      >
        {formError ? <FormAlert>{formError}</FormAlert> : null}

        <InfoSection icon={Landmark} title="Identitas">
          <div className="flex flex-col gap-4">
            <form.Field name="name">
              {(field) => (
                <TextField
                  field={field}
                  label="Nama kecamatan"
                  autoComplete="off"
                  onValueChange={(value) => {
                    if (!slugEdited.current) {
                      form.setFieldValue("slug", slugify(value));
                    }
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
                  onValueChange={() => {
                    slugEdited.current = true;
                  }}
                />
              )}
            </form.Field>
          </div>
        </InfoSection>

        <InfoSection icon={CloudRain} title="Info area">
          <div className="flex flex-col gap-4">
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
          </div>
        </InfoSection>
      </form>
    </FormSheet>
  );
}
