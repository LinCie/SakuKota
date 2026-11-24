import type { SubDistrict } from "~/features/subdistricts/types";
import { parseCsv } from "~/shared/csv";

import type { PlaceRequest } from "./api";
import { toPlaceRequest } from "./mappers";
import { attributeErrors, INDONESIA } from "./schemas";
import type { CategoryDetail, CategoryField } from "./types";
import { parseNumber, type AttributeValues, type PlaceValues } from "./values";

// Bulk import of one category's places from a CSV file, one place per row.

/** The most places one import may hold, as the backend allows (domain.MaxPlaceBatchSize). */
export const MAX_IMPORT_PLACES = 500;

/** A column of the import CSV and what it holds. */
export type PlaceCsvColumn = {
  name: string;
  hint: string;
  required: boolean;
};

const baseColumns: PlaceCsvColumn[] = [
  { name: "sub_district", hint: "Nama kecamatan.", required: true },
  { name: "name", hint: "Nama tempat, maks. 150 karakter.", required: true },
  {
    name: "latitude",
    hint: "Lintang, antara -11 dan 6. Contoh: -6.2088.",
    required: true,
  },
  {
    name: "longitude",
    hint: "Bujur, antara 95 dan 141. Contoh: 106.8456.",
    required: true,
  },
  { name: "address", hint: "Alamat, maks. 255 karakter.", required: false },
  {
    name: "operating_hours",
    hint: "Jam buka, maks. 255 karakter.",
    required: false,
  },
  {
    name: "estimated_price_range",
    hint: "Kisaran harga, maks. 100 karakter.",
    required: false,
  },
  { name: "short_description", hint: "Deskripsi singkat.", required: false },
];

/** Maximum lengths of the optional text columns, as the backend enforces. */
const maxLengths: Record<string, number> = {
  address: 255,
  operating_hours: 255,
  estimated_price_range: 100,
};

const booleans: Record<string, string> = {
  true: "true",
  ya: "true",
  false: "false",
  tidak: "false",
};

function fieldHint(field: CategoryField): string {
  switch (field.data_type) {
    case "number":
      return `${field.label}: angka.`;
    case "boolean":
      return `${field.label}: ya atau tidak.`;
    case "select":
      return `${field.label}: salah satu dari ${(field.options ?? []).join(", ")}.`;
    case "date":
      return `${field.label}: tanggal YYYY-MM-DD.`;
    default:
      return `${field.label}.`;
  }
}

/** The columns of the category's import CSV, in template order. A range field takes two. */
export function placeCsvColumns(category: CategoryDetail): PlaceCsvColumn[] {
  const columns = [...baseColumns];

  for (const field of category.fields) {
    if (field.data_type === "range") {
      columns.push(
        {
          name: `${field.key}_min`,
          hint: `${field.label}: batas bawah, angka.`,
          required: field.required,
        },
        {
          name: `${field.key}_max`,
          hint: `${field.label}: batas atas, angka.`,
          required: field.required,
        },
      );
    } else {
      columns.push({
        name: field.key,
        hint: fieldHint(field),
        required: field.required,
      });
    }
  }

  return columns;
}

/** The template file of the category's import: its header row only. */
export function placeCsvTemplate(category: CategoryDetail): string[][] {
  return [placeCsvColumns(category).map((column) => column.name)];
}

/** The places read from an import CSV, or every problem found in it. */
export type PlaceCsvResult =
  { ok: true; places: PlaceRequest[] } | { ok: false; errors: string[] };

/**
 * Reads the category's places from CSV text made from its template. Each row is
 * checked as the place form checks its input; errors name the spreadsheet row,
 * counting the header as row 1. Kecamatan are matched by name, ignoring case.
 */
export function readPlaceCsv(
  text: string,
  category: CategoryDetail,
  subDistricts: SubDistrict[],
): PlaceCsvResult {
  const records = parseCsv(text);
  const [header = [], ...rows] = records;
  const names = header.map((name) => name.trim().toLowerCase());
  const columns = placeCsvColumns(category).map((column) =>
    column.name.toLowerCase(),
  );

  const missing = columns.filter((column) => !names.includes(column));
  const unknown = names.filter((name) => !columns.includes(name));
  if (missing.length > 0 || unknown.length > 0) {
    const errors = [];
    if (missing.length > 0) {
      errors.push(`Kolom tidak ada: ${missing.join(", ")}.`);
    }
    if (unknown.length > 0) {
      errors.push(`Kolom tidak dikenal: ${unknown.join(", ")}.`);
    }
    errors.push(`Gunakan template kategori ${category.name}.`);
    return { ok: false, errors };
  }

  // Blank rows, such as trailing ones a spreadsheet leaves, are skipped.
  const filled = rows
    .map((cells, i) => ({ cells, row: i + 2 }))
    .filter(({ cells }) => cells.some((cell) => cell.trim() !== ""));
  if (filled.length === 0) {
    return { ok: false, errors: ["File tidak berisi baris tempat."] };
  }
  if (filled.length > MAX_IMPORT_PLACES) {
    return {
      ok: false,
      errors: [`Maksimal ${MAX_IMPORT_PLACES} tempat sekali impor.`],
    };
  }

  const kecamatan = new Map(
    subDistricts.map((s) => [s.name.trim().toLowerCase(), s]),
  );
  const places: PlaceRequest[] = [];
  const errors: string[] = [];

  for (const { cells, row } of filled) {
    const cell = (column: string) =>
      (cells[names.indexOf(column.toLowerCase())] ?? "").trim();
    const rowErrors: string[] = [];

    const subDistrictName = cell("sub_district");
    const subDistrict = kecamatan.get(subDistrictName.toLowerCase());
    if (!subDistrict) {
      rowErrors.push(
        subDistrictName
          ? `kecamatan tidak dikenal: ${subDistrictName}.`
          : "sub_district wajib diisi.",
      );
    }

    const name = cell("name");
    if (name === "") rowErrors.push("name wajib diisi.");
    else if ([...name].length > 150) {
      rowErrors.push("name maksimal 150 karakter.");
    }

    const lat = parseNumber(cell("latitude"));
    if (!(lat >= INDONESIA.minLat && lat <= INDONESIA.maxLat)) {
      rowErrors.push("latitude harus angka antara -11 dan 6.");
    }
    const lng = parseNumber(cell("longitude"));
    if (!(lng >= INDONESIA.minLng && lng <= INDONESIA.maxLng)) {
      rowErrors.push("longitude harus angka antara 95 dan 141.");
    }

    for (const [column, max] of Object.entries(maxLengths)) {
      if ([...cell(column)].length > max) {
        rowErrors.push(`${column} maksimal ${max} karakter.`);
      }
    }

    const attributes: AttributeValues = {};
    // Errors by field key, so a field is reported once.
    const fieldErrors: Record<string, string> = {};
    for (const field of category.fields) {
      if (field.data_type === "range") {
        attributes[field.key] = {
          min: cell(`${field.key}_min`),
          max: cell(`${field.key}_max`),
        };
        continue;
      }

      const value = cell(field.key);
      attributes[field.key] = value;
      if (value === "") continue;

      if (field.data_type === "boolean") {
        const normalized = booleans[value.toLowerCase()];
        if (normalized) attributes[field.key] = normalized;
        else fieldErrors[field.key] = "Isi dengan ya atau tidak.";
      }
      if (field.data_type === "select" && !field.options?.includes(value)) {
        fieldErrors[field.key] =
          `Isi dengan salah satu dari ${(field.options ?? []).join(", ")}.`;
      }
    }
    const merged = {
      ...attributeErrors(category.fields, attributes),
      ...fieldErrors,
    };
    for (const [key, message] of Object.entries(merged)) {
      rowErrors.push(`${key}: ${message}`);
    }

    if (rowErrors.length > 0) {
      errors.push(...rowErrors.map((error) => `Baris ${row}: ${error}`));
      continue;
    }

    const values: PlaceValues = {
      subDistrictId: subDistrict ? String(subDistrict.id) : "",
      categoryId: String(category.id),
      name,
      lat: cell("latitude"),
      lng: cell("longitude"),
      address: cell("address"),
      operatingHours: cell("operating_hours"),
      priceRange: cell("estimated_price_range"),
      description: cell("short_description"),
      attributes,
    };
    places.push(toPlaceRequest(values, category.fields));
  }

  return errors.length > 0 ? { ok: false, errors } : { ok: true, places };
}
