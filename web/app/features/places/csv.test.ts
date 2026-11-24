import { describe, expect, it } from "vitest";

import type { SubDistrict } from "~/features/subdistricts/types";
import { toCsv } from "~/shared/csv";

import { placeCsvColumns, placeCsvTemplate, readPlaceCsv } from "./csv";
import { category, field } from "./fixtures";
import type { CategoryDetail } from "./types";

const warung: CategoryDetail = {
  ...category,
  fields: [
    field({ key: "halal", data_type: "boolean" }),
    field({
      key: "serving_style",
      data_type: "select",
      required: true,
      options: ["Prasmanan", "Dimasak saat dipesan"],
    }),
    field({ key: "price", data_type: "range" }),
    field({ key: "seats", data_type: "number" }),
  ],
};

function subDistrict(id: number, name: string): SubDistrict {
  return {
    id,
    name,
    slug: name.toLowerCase(),
    flood_history: null,
    safety_crime_information: null,
    estimated_daily_spending: null,
    medias: [],
    boundary: null,
  };
}

const subDistricts = [subDistrict(7, "Setiabudi"), subDistrict(8, "Menteng")];

const header = placeCsvTemplate(warung)[0];

/** CSV text of the template header and rows given as column → value. */
function csv(...rows: Record<string, string>[]): string {
  return toCsv([header, ...rows.map((row) => header.map((c) => row[c] ?? ""))]);
}

const validRow = {
  sub_district: "setiabudi",
  name: " Warung Bu Sri ",
  latitude: "-6,2088",
  longitude: "106.8456",
  address: "Jl. Melati 5",
  serving_style: "Prasmanan",
};

describe("placeCsvColumns", () => {
  it("lists the place columns, then one per field and two per range", () => {
    expect(placeCsvColumns(warung).map((c) => c.name)).toEqual([
      "sub_district",
      "name",
      "latitude",
      "longitude",
      "address",
      "operating_hours",
      "estimated_price_min",
      "estimated_price_max",
      "short_description",
      "halal",
      "serving_style",
      "price_min",
      "price_max",
      "seats",
    ]);
  });
});

describe("placeCsvColumns for a category without prices", () => {
  const masjid: CategoryDetail = { ...category, has_price: false, fields: [] };

  it("has no price columns", () => {
    const names = placeCsvColumns(masjid).map((c) => c.name);
    expect(names).toContain("operating_hours");
    expect(names).not.toContain("estimated_price_min");
    expect(names).not.toContain("estimated_price_max");
  });

  it("reads a row without a price", () => {
    const text = toCsv([
      placeCsvTemplate(masjid)[0],
      placeCsvTemplate(masjid)[0].map(
        (c) =>
          ({
            sub_district: "Setiabudi",
            name: "Masjid Al-Ikhlas",
            latitude: "-6.2",
            longitude: "106.8",
            operating_hours: "Setiap hari, 04:00-21:30",
          })[c] ?? "",
      ),
    ]);

    const result = readPlaceCsv(text, masjid, subDistricts);
    expect(result.ok && result.places[0]).toMatchObject({
      estimated_price_range: null,
      operating_hours: [1, 2, 3, 4, 5, 6, 7].map((day) => ({
        day,
        open: "04:00",
        close: "21:30",
      })),
    });
  });
});

describe("readPlaceCsv", () => {
  it("reads rows into place requests of the category", () => {
    const result = readPlaceCsv(
      csv(validRow, {
        ...validRow,
        sub_district: "Menteng",
        name: "Warung Pak Budi",
        halal: "Ya",
        price_min: "10000",
        price_max: "25000",
        seats: "12",
      }),
      warung,
      subDistricts,
    );

    expect(result).toEqual({
      ok: true,
      places: [
        {
          sub_district_id: 7,
          category_id: warung.id,
          name: "Warung Bu Sri",
          address: "Jl. Melati 5",
          location: { type: "Point", coordinates: [106.8456, -6.2088] },
          operating_hours: null,
          estimated_price_range: null,
          short_description: null,
          attributes: { serving_style: "Prasmanan" },
        },
        {
          sub_district_id: 8,
          category_id: warung.id,
          name: "Warung Pak Budi",
          address: "Jl. Melati 5",
          location: { type: "Point", coordinates: [106.8456, -6.2088] },
          operating_hours: null,
          estimated_price_range: null,
          short_description: null,
          attributes: {
            halal: true,
            serving_style: "Prasmanan",
            price: { min: 10000, max: 25000 },
            seats: 12,
          },
        },
      ],
    });
  });

  it("reads hours and a price range", () => {
    const result = readPlaceCsv(
      csv(
        {
          ...validRow,
          operating_hours:
            "Sen-Jum, 08:00-20:00; Sab, 08:00-12:00, 16:00-20:00",
          estimated_price_min: "15000",
          estimated_price_max: "25000",
        },
        { ...validRow, estimated_price_min: "3500" },
      ),
      warung,
      subDistricts,
    );

    expect(result.ok && result.places.map((p) => p.operating_hours)).toEqual([
      [
        ...[1, 2, 3, 4, 5].map((day) => ({
          day,
          open: "08:00",
          close: "20:00",
        })),
        { day: 6, open: "08:00", close: "12:00" },
        { day: 6, open: "16:00", close: "20:00" },
      ],
      null,
    ]);
    expect(
      result.ok && result.places.map((p) => p.estimated_price_range),
    ).toEqual([
      { min: 15000, max: 25000 },
      { min: 3500, max: 3500 },
    ]);
  });

  it("reports hours and prices that cannot be read", () => {
    const result = readPlaceCsv(
      csv(
        { ...validRow, operating_hours: "kapan saja" },
        { ...validRow, operating_hours: "Sen, 20:00-08:00" },
        { ...validRow, operating_hours: "Sen, 08:00-12:00, 11:00-14:00" },
        { ...validRow, estimated_price_min: "9", estimated_price_max: "1" },
        { ...validRow, estimated_price_min: "15.000" },
        { ...validRow, estimated_price_max: "murah" },
      ),
      warung,
      subDistricts,
    );

    expect(result.ok).toBe(false);
    const errors = result.ok ? [] : result.errors;
    expect(errors).toHaveLength(6);
    expect(errors[0]).toMatch(/^Baris 2: operating_hours: /);
    expect(errors[1]).toMatch(
      /^Baris 3: operating_hours: Senin: jam buka harus sebelum/,
    );
    expect(errors[2]).toMatch(
      /^Baris 4: operating_hours: Senin: jam buka saling tumpang/,
    );
    expect(errors[3]).toMatch(/^Baris 5: estimated_price_min: Harga terendah/);
    expect(errors[4]).toMatch(
      /^Baris 6: estimated_price_min: Isi rupiah bulat/,
    );
    expect(errors[5]).toMatch(
      /^Baris 7: estimated_price_max: Isi rupiah bulat/,
    );
  });

  it("accepts columns in any order and skips blank rows", () => {
    const columns = [...header].reverse();
    const text = toCsv([
      columns,
      columns.map((c) => validRow[c as keyof typeof validRow] ?? ""),
      columns.map(() => ""),
    ]);

    const result = readPlaceCsv(text, warung, subDistricts);
    expect(result.ok && result.places.map((p) => p.name)).toEqual([
      "Warung Bu Sri",
    ]);
  });

  it("rejects a header that is not the category's template", () => {
    const text = toCsv([[...header.filter((c) => c !== "seats"), "wifi"], []]);

    expect(readPlaceCsv(text, warung, subDistricts)).toEqual({
      ok: false,
      errors: [
        "Kolom tidak ada: seats.",
        "Kolom tidak dikenal: wifi.",
        `Gunakan template kategori ${warung.name}.`,
      ],
    });
  });

  it("rejects a file without places", () => {
    expect(readPlaceCsv(csv(), warung, subDistricts)).toEqual({
      ok: false,
      errors: ["File tidak berisi baris tempat."],
    });
  });

  it("rejects more than 500 places", () => {
    const rows = Array.from({ length: 501 }, () => validRow);
    const result = readPlaceCsv(csv(...rows), warung, subDistricts);
    expect(result.ok).toBe(false);
  });

  it("names the spreadsheet row of every problem", () => {
    const result = readPlaceCsv(
      csv(validRow, {
        sub_district: "Tebet",
        name: "",
        latitude: "abc",
        longitude: "200",
        halal: "mungkin",
        serving_style: "Bungkus",
        price_min: "5",
        price_max: "1",
        seats: "banyak",
      }),
      warung,
      subDistricts,
    );

    expect(result).toEqual({
      ok: false,
      errors: [
        "Baris 3: kecamatan tidak dikenal: Tebet.",
        "Baris 3: name wajib diisi.",
        "Baris 3: latitude harus angka antara -11 dan 6.",
        "Baris 3: longitude harus angka antara 95 dan 141.",
        "Baris 3: price: Batas bawah tidak boleh melebihi batas atas.",
        "Baris 3: seats: Isi dengan angka.",
        "Baris 3: halal: Isi dengan ya atau tidak.",
        "Baris 3: serving_style: Isi dengan salah satu dari Prasmanan, Dimasak saat dipesan.",
      ],
    });
  });

  it("requires required fields", () => {
    const result = readPlaceCsv(
      csv({ ...validRow, serving_style: "" }),
      warung,
      subDistricts,
    );
    expect(result).toEqual({
      ok: false,
      errors: ["Baris 2: serving_style: Wajib diisi."],
    });
  });
});
