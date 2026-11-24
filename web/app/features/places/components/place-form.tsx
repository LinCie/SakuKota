import { useSelector } from "@tanstack/react-form";
import { LocateFixed } from "lucide-react";
import type { ReactNode } from "react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { Button } from "~/shared/components/ui/button";
import {
  SelectField,
  TextAreaField,
  TextField,
} from "~/shared/components/field";
import { FormSection } from "~/shared/components/form-section";
import { useCurrentPosition } from "~/shared/hooks";
import type { AppFormApi } from "~/shared/form-api";

import { attributeErrors, validCoordinate } from "../schemas";
import type { CategoryDetail, Place } from "../types";
import { parseNumber, type PlaceValues } from "../values";
import { CategoryFieldInput } from "./category-field-input";
import { LocationPicker, type Pin } from "./location-picker";
import { OperatingHoursInput } from "./operating-hours-input";

type PlaceFormProps = {
  form: AppFormApi<PlaceValues>;
  formId: string;
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  /** Every place, so the map can show what the chosen kecamatan already has. */
  places: Place[];
  /** Extra sections after the place's own, inside the same form. */
  children?: ReactNode;
};

/** The fields of the place form; the drawer owns the form and its submit. */
export function PlaceForm({
  form,
  formId,
  categories,
  subDistricts,
  places,
  children,
}: PlaceFormProps) {
  const subDistrictId = useSelector(
    form.store,
    (state) => state.values.subDistrictId,
  );
  const categoryId = useSelector(
    form.store,
    (state) => state.values.categoryId,
  );
  const lat = useSelector(form.store, (state) => state.values.lat);
  const lng = useSelector(form.store, (state) => state.values.lng);
  const position = useCurrentPosition();

  const category = categories.find((c) => String(c.id) === categoryId);
  const neighbours = places.filter(
    (place) =>
      String(place.sub_district_id) === subDistrictId && place.location,
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
              // Each category has its own custom fields, and only some have prices.
              onValueChange={(value) => {
                form.setFieldValue("attributes", {});
                if (
                  !categories.find((c) => String(c.id) === value)?.has_price
                ) {
                  form.setFieldValue("priceMin", "");
                  form.setFieldValue("priceMax", "");
                }
              }}
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
          <p className="text-muted-foreground text-sm text-pretty">
            Ketuk peta untuk menanam tanda, lalu seret untuk menggesernya.
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <Button
            variant="outline"
            onClick={() => position.locate(pick)}
            disabled={position.locating}
            className="self-start"
          >
            <LocateFixed aria-hidden="true" data-icon="inline-start" />
            {position.locating ? "Mencari lokasi…" : "Pakai lokasi saya"}
          </Button>
          {position.error ? (
            <p role="status" className="text-destructive text-sm">
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
        <form.Field name="hours">
          {(field) => (
            <OperatingHoursInput
              value={field.state.value}
              onChange={field.handleChange}
              error={field.state.meta.errors[0]?.message}
            />
          )}
        </form.Field>
        {category?.has_price ? (
          <div className="flex flex-col gap-1.5">
            <span
              id="price-label"
              className="text-foreground text-sm font-medium"
            >
              Kisaran harga
              {category.price_unit ? ` per ${category.price_unit}` : ""}
              <span className="text-muted-foreground ml-1.5 font-normal">
                opsional
              </span>
            </span>
            <div
              role="group"
              aria-labelledby="price-label"
              className="grid grid-cols-2 gap-3"
            >
              <form.Field name="priceMin">
                {(field) => (
                  <TextField
                    field={field}
                    label="Terendah"
                    prefix="Rp"
                    inputMode="numeric"
                    autoComplete="off"
                    placeholder="15000"
                  />
                )}
              </form.Field>
              <form.Field name="priceMax">
                {(field) => (
                  <TextField
                    field={field}
                    label="Tertinggi"
                    prefix="Rp"
                    inputMode="numeric"
                    autoComplete="off"
                    placeholder="25000"
                  />
                )}
              </form.Field>
            </div>
            <p className="text-muted-foreground text-sm">
              Isi hanya Terendah untuk harga tunggal. Rupiah bulat, tanpa titik.
            </p>
          </div>
        ) : null}
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

      {children}
    </form>
  );
}
