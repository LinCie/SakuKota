import { useStore } from "@tanstack/react-form";
import { LocateFixed } from "lucide-react";
import { useMemo } from "react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { Button } from "~/shared/components/button";
import {
  SelectField,
  TextAreaField,
  TextField,
} from "~/shared/components/form-fields";
import { FormSection } from "~/shared/components/form-section";
import { useCurrentPosition } from "~/shared/hooks/use-current-position";
import type { AppFormApi } from "~/shared/form-api";

import { attributeErrors, parseNumber, validCoordinate } from "../schemas";
import type { PlaceValues } from "../place-values";
import type { CategoryDetail, Place } from "../types";
import { CategoryFieldInput } from "./category-field-input";
import { LocationPicker, type Pin } from "./location-picker";

type PlaceFormProps = {
  form: AppFormApi<PlaceValues>;
  formId: string;
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  /** Every place, so the map can show what the chosen kecamatan already has. */
  places: Place[];
};

/** The fields of the place form; the drawer owns the form and its submit. */
export function PlaceForm({
  form,
  formId,
  categories,
  subDistricts,
  places,
}: PlaceFormProps) {
  const subDistrictId = useStore(
    form.store,
    (state) => state.values.subDistrictId,
  );
  const categoryId = useStore(form.store, (state) => state.values.categoryId);
  const lat = useStore(form.store, (state) => state.values.lat);
  const lng = useStore(form.store, (state) => state.values.lng);
  const position = useCurrentPosition();

  const category = categories.find((c) => String(c.id) === categoryId);
  const neighbours = useMemo(
    () =>
      places.filter(
        (place) =>
          String(place.sub_district_id) === subDistrictId && place.location,
      ),
    [places, subDistrictId],
  );
  const pin: Pin | null =
    validCoordinate(lat, 90) && validCoordinate(lng, 180)
      ? { lat: parseNumber(lat), lng: parseNumber(lng) }
      : null;

  function pick(next: Pin) {
    form.setFieldValue("lat", String(next.lat));
    form.setFieldValue("lng", String(next.lng));
  }

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
      <FormSection title="Dasar">
        <form.Field name="subDistrictId">
          {(field) => (
            <SelectField
              field={field}
              label="Kecamatan"
              placeholder="Pilih kecamatan"
              options={subDistricts.map((sd) => ({
                value: String(sd.id),
                label: `Kec. ${sd.name}`,
              }))}
            />
          )}
        </form.Field>
        <form.Field name="categoryId">
          {(field) => (
            <SelectField
              field={field}
              label="Kategori"
              placeholder="Pilih kategori"
              options={categories.map((c) => ({
                value: String(c.id),
                label: c.name,
              }))}
              // Each category has its own custom fields.
              onValueChange={() => form.setFieldValue("attributes", {})}
            />
          )}
        </form.Field>
        <form.Field name="name">
          {(field) => (
            <TextField field={field} label="Nama tempat" autoComplete="off" />
          )}
        </form.Field>
      </FormSection>

      <FormSection title="Lokasi">
        <div className="flex flex-col gap-2">
          <LocationPicker
            pin={pin}
            onPick={pick}
            neighbours={neighbours}
            category={category}
            focusKey={subDistrictId}
          />
          <p className="text-ink-soft text-sm text-pretty">
            Ketuk peta untuk menanam tanda, lalu seret untuk menggesernya.
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <Button
            onClick={() => position.locate(pick)}
            disabled={position.locating}
            className="self-start"
          >
            <LocateFixed
              aria-hidden="true"
              className="size-4"
              strokeWidth={2.5}
            />
            {position.locating ? "Mencari lokasi…" : "Pakai lokasi saya"}
          </Button>
          {position.error ? (
            <p role="status" className="text-chili text-sm font-semibold">
              {position.error}
            </p>
          ) : null}
        </div>

        <div className="grid grid-cols-2 gap-3">
          <form.Field name="lat">
            {(field) => (
              <TextField
                field={field}
                label="Lintang"
                inputMode="decimal"
                autoComplete="off"
                placeholder="-6.2088"
              />
            )}
          </form.Field>
          <form.Field name="lng">
            {(field) => (
              <TextField
                field={field}
                label="Bujur"
                inputMode="decimal"
                autoComplete="off"
                placeholder="106.8456"
              />
            )}
          </form.Field>
        </div>

        <form.Field name="address">
          {(field) => (
            <TextAreaField field={field} label="Alamat" optional rows={2} />
          )}
        </form.Field>
      </FormSection>

      <FormSection title="Detail">
        <form.Field name="operatingHours">
          {(field) => (
            <TextField
              field={field}
              label="Jam buka"
              optional
              hint="Contoh: 07.00–21.00"
              autoComplete="off"
            />
          )}
        </form.Field>
        <form.Field name="priceRange">
          {(field) => (
            <TextField
              field={field}
              label="Kisaran harga"
              optional
              hint="Contoh: Rp15.000–25.000"
              autoComplete="off"
            />
          )}
        </form.Field>
        <form.Field name="description">
          {(field) => (
            <TextAreaField field={field} label="Deskripsi singkat" optional />
          )}
        </form.Field>
      </FormSection>

      {category && category.fields.length > 0 ? (
        <FormSection title={`Khusus ${category.name}`}>
          <form.Field
            name="attributes"
            validators={{
              onSubmit: ({ value }) => {
                const errors = attributeErrors(category.fields, value);
                return Object.keys(errors).length > 0 ? errors : undefined;
              },
            }}
          >
            {(field) => {
              // Set by this field's own validator above.
              const errors = (field.state.meta.errors[0] ?? {}) as Record<
                string,
                string
              >;
              return (
                <div className="flex flex-col gap-4">
                  {category.fields.map((categoryField) => (
                    <CategoryFieldInput
                      key={categoryField.key}
                      field={categoryField}
                      value={field.state.value[categoryField.key]}
                      onChange={(next) =>
                        field.handleChange({
                          ...field.state.value,
                          [categoryField.key]: next,
                        })
                      }
                      error={errors[categoryField.key]}
                    />
                  ))}
                </div>
              );
            }}
          </form.Field>
        </FormSection>
      ) : null}
    </form>
  );
}
