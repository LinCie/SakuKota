import { useForm, useStore } from "@tanstack/react-form";
import {
  ClipboardList,
  ListChecks,
  LocateFixed,
  MapPin,
  Store,
} from "lucide-react";
import { useMemo, useState } from "react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { adminWriteError } from "~/shared/admin-error";
import {
  FormAlert,
  SelectField,
  TextAreaField,
  TextField,
} from "~/shared/components/form-fields";
import { FormSheet } from "~/shared/components/form-sheet";
import { InfoSection } from "~/shared/components/menu-board";
import { createPlace } from "../api";
import {
  attributeErrors,
  buildAttributes,
  parseNumber,
  placeSchema,
  validCoordinate,
  type AttributeValues,
} from "../schemas";
import type { CategoryDetail, Place } from "../types";
import { CategoryFieldInput } from "./category-field-input";
import { LocationPicker, type Pin } from "./location-picker";

type PlaceValues = {
  subDistrictId: string;
  categoryId: string;
  name: string;
  lat: string;
  lng: string;
  address: string;
  operatingHours: string;
  priceRange: string;
  description: string;
  attributes: AttributeValues;
};

const emptyValues: PlaceValues = {
  subDistrictId: "",
  categoryId: "",
  name: "",
  lat: "",
  lng: "",
  address: "",
  operatingHours: "",
  priceRange: "",
  description: "",
  attributes: {},
};

function orNull(text: string) {
  const trimmed = text.trim();
  return trimmed === "" ? null : trimmed;
}

type PlaceSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  /** Every place, so the map can show what the chosen kecamatan already has. */
  places: Place[];
  /** Called with the new place once it is saved, before the sheet closes. */
  onCreated: (place: Place) => Promise<void>;
};

/** The "Tambah tempat" form: a sheet with a map to plant the place's sign. */
export function PlaceSheet({
  open,
  onOpenChange,
  categories,
  subDistricts,
  places,
  onCreated,
}: PlaceSheetProps) {
  const [formError, setFormError] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);
  const [locateError, setLocateError] = useState<string | null>(null);
  // Batch entry: the next place usually shares the last one's kecamatan and category.
  const [last, setLast] = useState({ subDistrictId: "", categoryId: "" });

  function startValues(): PlaceValues {
    return {
      ...emptyValues,
      subDistrictId:
        last.subDistrictId ||
        (subDistricts.length === 1 ? String(subDistricts[0].id) : ""),
      categoryId: last.categoryId,
    };
  }

  const form = useForm({
    defaultValues: startValues(),
    validators: { onSubmit: placeSchema },
    onSubmit: async ({ value }) => {
      setFormError(null);
      const category = categories.find(
        (c) => String(c.id) === value.categoryId,
      );

      try {
        const result = await createPlace({
          sub_district_id: Number(value.subDistrictId),
          category_id: Number(value.categoryId),
          name: value.name.trim(),
          address: orNull(value.address),
          location: {
            type: "Point",
            coordinates: [parseNumber(value.lng), parseNumber(value.lat)],
          },
          operating_hours: orNull(value.operatingHours),
          estimated_price_range: orNull(value.priceRange),
          short_description: orNull(value.description),
          attributes: buildAttributes(category?.fields ?? [], value.attributes),
        });
        if (!result.ok) {
          setFormError(adminWriteError(result.status, result.error));
          return;
        }
        setLast({
          subDistrictId: value.subDistrictId,
          categoryId: value.categoryId,
        });
        await onCreated(result.data);
        onOpenChange(false);
      } catch {
        setFormError("Tidak bisa terhubung ke server. Coba lagi.");
      }
    },
  });

  const pending = useStore(form.store, (state) => state.isSubmitting);
  const dirty = useStore(form.store, (state) => state.isDirty);
  const subDistrictId = useStore(form.store, (s) => s.values.subDistrictId);
  const categoryId = useStore(form.store, (s) => s.values.categoryId);
  const lat = useStore(form.store, (s) => s.values.lat);
  const lng = useStore(form.store, (s) => s.values.lng);

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
    setLocateError(null);
  }

  function locateMe() {
    setLocateError(null);
    if (!("geolocation" in navigator)) {
      setLocateError("Perangkat ini tidak mendukung lokasi. Ketuk peta saja.");
      return;
    }
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        setLocating(false);
        pick({ lat: coords.latitude, lng: coords.longitude });
      },
      () => {
        setLocating(false);
        setLocateError(
          "Lokasimu tidak terbaca. Izinkan lokasi, atau ketuk peta.",
        );
      },
      { enableHighAccuracy: true, timeout: 10_000 },
    );
  }

  return (
    <FormSheet
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(startValues());
        setFormError(null);
        setLocateError(null);
      }}
      title="Tambah tempat"
      dirty={dirty}
      formId="place-form"
      pending={pending}
      submitLabel="Simpan tempat"
    >
      <form
        id="place-form"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
        className="flex flex-col gap-5"
      >
        {formError ? <FormAlert>{formError}</FormAlert> : null}

        <InfoSection icon={Store} title="Dasar">
          <div className="flex flex-col gap-4">
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
                <TextField
                  field={field}
                  label="Nama tempat"
                  autoComplete="off"
                />
              )}
            </form.Field>
          </div>
        </InfoSection>

        <InfoSection icon={MapPin} title="Lokasi">
          <div className="flex flex-col gap-4">
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
                Tanda kecil abu-abu adalah tempat yang sudah ada.
              </p>
            </div>

            <div className="flex flex-col gap-2">
              <button
                type="button"
                onClick={locateMe}
                disabled={locating}
                className="border-ink/60 bg-paint text-ink hover:border-ink hover:bg-sign-soft focus-visible:outline-cat inline-flex h-11 items-center justify-center gap-2 rounded-[3px] border-2 border-dashed px-4 text-sm font-bold transition-colors hover:border-solid focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-50"
              >
                <LocateFixed
                  aria-hidden="true"
                  className="size-4"
                  strokeWidth={2.5}
                />
                {locating ? "Mencari lokasi…" : "Pakai lokasi saya"}
              </button>
              {locateError ? (
                <p role="status" className="text-chili text-sm font-semibold">
                  {locateError}
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
                <TextAreaField
                  field={field}
                  label="Alamat"
                  optional
                  rows={2}
                  autoComplete="off"
                />
              )}
            </form.Field>
          </div>
        </InfoSection>

        <InfoSection icon={ClipboardList} title="Detail">
          <div className="flex flex-col gap-4">
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
                <TextAreaField
                  field={field}
                  label="Deskripsi singkat"
                  optional
                />
              )}
            </form.Field>
          </div>
        </InfoSection>

        {category && category.fields.length > 0 ? (
          <InfoSection icon={ListChecks} title={`Khusus ${category.name}`}>
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
          </InfoSection>
        ) : null}
      </form>
    </FormSheet>
  );
}
