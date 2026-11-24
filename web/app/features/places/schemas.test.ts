import { describe, expect, it } from "vitest";

import { field } from "./fixtures";
import { emptyHours } from "./hours";
import { attributeErrors, placeSchema, validCoordinate } from "./schemas";
import { emptyPlaceValues, type PlaceValues } from "./values";

describe("attributeErrors", () => {
  const required = (data_type: string) =>
    field({ key: "k", data_type, required: true });
  const optional = (data_type: string) =>
    field({ key: "k", data_type, required: false });

  describe("required", () => {
    it.each([
      ["missing", {}],
      ["empty", { k: "" }],
      ["whitespace", { k: "   " }],
    ])("rejects a %s text field", (_, values) => {
      expect(attributeErrors([required("text")], values)).toEqual({
        k: "Wajib diisi.",
      });
    });

    it("rejects an unanswered yes-no field", () => {
      expect(attributeErrors([required("boolean")], { k: "" })).toHaveProperty(
        "k",
      );
    });

    it("accepts 'false' and '0' as answers", () => {
      expect(attributeErrors([required("boolean")], { k: "false" })).toEqual(
        {},
      );
      expect(attributeErrors([required("number")], { k: "0" })).toEqual({});
    });

    it("rejects a required range with both ends empty", () => {
      expect(
        attributeErrors([required("range")], { k: { min: "", max: "" } }),
      ).toHaveProperty("k", "Wajib diisi.");
    });
  });

  describe("optional", () => {
    it.each(["text", "number", "range", "boolean", "date"])(
      "ignores a blank %s field",
      (type) => {
        expect(attributeErrors([optional(type)], {})).toEqual({});
        expect(attributeErrors([optional(type)], { k: "  " })).toEqual({});
      },
    );
  });

  describe("number", () => {
    it.each(["abc", "1,5,3", "12 000", "-"])("rejects %j", (text) => {
      expect(attributeErrors([optional("number")], { k: text })).toEqual({
        k: "Isi dengan angka.",
      });
    });

    it.each(["1,5", "0", "-3", "  7  "])("accepts %j", (text) => {
      expect(attributeErrors([optional("number")], { k: text })).toEqual({});
    });

    // JSON.stringify(Infinity) is "null", so these must not reach the request.
    it.each(["Infinity", "-Infinity"])("rejects %j", (text) => {
      expect(attributeErrors([optional("number")], { k: text })).toHaveProperty(
        "k",
      );
    });
  });

  describe("range", () => {
    const check = (min: string, max: string) =>
      attributeErrors([optional("range")], { k: { min, max } });

    it("rejects a half-filled range", () => {
      const message = "Isi batas bawah dan batas atas dengan angka.";
      expect(check("5", "")).toEqual({ k: message });
      expect(check("", "5")).toEqual({ k: message });
    });

    it("rejects an end that is not a number", () => {
      expect(check("abc", "5")).toHaveProperty("k");
      expect(check("5", "abc")).toHaveProperty("k");
    });

    it("rejects min above max", () => {
      expect(check("10", "5")).toEqual({
        k: "Batas bawah tidak boleh melebihi batas atas.",
      });
    });

    it("compares numerically, not as text", () => {
      expect(check("9", "10")).toEqual({});
      expect(check("10", "9")).toHaveProperty("k");
    });

    it("accepts equal ends, zero, negatives and decimal commas", () => {
      expect(check("5", "5")).toEqual({});
      expect(check("0", "0")).toEqual({});
      expect(check("-5", "-1")).toEqual({});
      expect(check("1,5", "2,5")).toEqual({});
    });

    it("rejects infinite ends", () => {
      expect(check("0", "Infinity")).toHaveProperty("k");
    });
  });

  describe("other types", () => {
    it("does not throw for a data type it does not know", () => {
      expect(attributeErrors([optional("color")], { k: "red" })).toEqual({});
      expect(attributeErrors([required("color")], { k: "" })).toHaveProperty(
        "k",
      );
    });

    it("does not validate text or select values beyond required", () => {
      expect(attributeErrors([optional("select")], { k: "anything" })).toEqual(
        {},
      );
    });

    // The backend parses YYYY-MM-DD strictly (time.DateOnly).
    it.each([
      "not-a-date",
      "2025-02-30",
      "2025-13-01",
      "2025-1-1",
      "04/03/2025",
      "2025-03-04T00:00:00Z",
    ])("rejects the date %j", (text) => {
      expect(attributeErrors([optional("date")], { k: text })).toHaveProperty(
        "k",
      );
    });

    it.each(["2025-03-04", "2024-02-29"])("accepts the date %j", (text) => {
      expect(attributeErrors([optional("date")], { k: text })).toEqual({});
    });
  });

  it("reports every invalid field, keyed by field", () => {
    const fields = [
      field({ key: "a", data_type: "number" }),
      field({ key: "b", required: true }),
      field({ key: "c" }),
    ];
    expect(Object.keys(attributeErrors(fields, { a: "x" })).sort()).toEqual([
      "a",
      "b",
    ]);
  });

  it("ignores values for fields the category does not have", () => {
    expect(attributeErrors([], { stale: "x" })).toEqual({});
  });
});

describe("validCoordinate", () => {
  it.each([
    ["0", true],
    ["90", true],
    ["-90", true],
    ["-6,2", true],
    ["90.0001", false],
    ["-90.0001", false],
    ["", false],
    ["abc", false],
    ["Infinity", false],
  ])("%j within 90 → %s", (text, expected) => {
    expect(validCoordinate(text, 90)).toBe(expected);
  });
});

describe("placeSchema", () => {
  const valid = (): PlaceValues => ({
    ...emptyPlaceValues([]),
    subDistrictId: "7",
    categoryId: "3",
    name: "Warung Bu Tini",
    lat: "-6.2",
    lng: "106.8",
  });

  const messages = (values: PlaceValues, path: string) => {
    const result = placeSchema.safeParse(values);
    if (result.success) return [];
    return result.error.issues
      .filter((issue) => issue.path.join(".") === path)
      .map((issue) => issue.message);
  };

  it("accepts a valid place", () => {
    expect(placeSchema.safeParse(valid()).success).toBe(true);
  });

  describe("hours and price", () => {
    it("reports an hours problem on hours", () => {
      const hours = emptyHours();
      hours[2] = {
        mode: "hours",
        periods: [{ open: "20:00", close: "08:00" }],
      };
      const [message] = messages({ ...valid(), hours }, "hours");
      expect(message).toMatch(/^Rabu: jam buka harus sebelum jam tutup/);
    });

    it("accepts hours and a price range", () => {
      const hours = emptyHours();
      hours[0] = { mode: "24h", periods: [] };
      const values = { ...valid(), hours, priceMin: "9000", priceMax: "12000" };
      expect(placeSchema.safeParse(values).success).toBe(true);
    });

    it("reports a price problem on the input it belongs to", () => {
      expect(
        messages({ ...valid(), priceMin: "9,5" }, "priceMin"),
      ).toHaveLength(1);
      expect(
        messages({ ...valid(), priceMin: "9", priceMax: "1" }, "priceMin"),
      ).toEqual(["Harga terendah tidak boleh melebihi harga tertinggi."]);
      expect(
        messages({ ...valid(), priceMax: "abc" }, "priceMax"),
      ).toHaveLength(1);
    });
  });

  it("requires a kecamatan and a category", () => {
    const values = { ...valid(), subDistrictId: "", categoryId: "" };
    expect(messages(values, "subDistrictId")).toEqual(["Pilih kecamatan."]);
    expect(messages(values, "categoryId")).toEqual(["Pilih kategori."]);
  });

  describe("name", () => {
    it.each(["", "   ", "\t\n"])("rejects %j", (name) => {
      expect(messages({ ...valid(), name }, "name")).toEqual([
        "Nama tempat wajib diisi.",
      ]);
    });

    it("allows 150 characters and rejects 151", () => {
      expect(messages({ ...valid(), name: "a".repeat(150) }, "name")).toEqual(
        [],
      );
      expect(messages({ ...valid(), name: "a".repeat(151) }, "name")).toEqual([
        "Nama tempat maksimal 150 karakter.",
      ]);
    });

    it("does not count surrounding spaces toward the limit", () => {
      const name = `  ${"a".repeat(150)}  `;
      expect(messages({ ...valid(), name }, "name")).toEqual([]);
    });
  });

  describe("coordinates", () => {
    it.each([
      ["lat", "-11"],
      ["lat", "6"],
      ["lat", "-6,2"],
      ["lng", "95"],
      ["lng", "141"],
      ["lng", "106,8"],
    ])("accepts %s = %s", (key, text) => {
      expect(messages({ ...valid(), [key]: text }, key)).toEqual([]);
    });

    it.each([
      ["lat", "-11.0001"],
      ["lat", "6.0001"],
      ["lat", "40"],
      ["lng", "94.9999"],
      ["lng", "141.0001"],
      ["lng", "-106.8"],
    ])("rejects %s = %s as outside Indonesia", (key, text) => {
      const [message] = messages({ ...valid(), [key]: text }, key);
      expect(message).toMatch(/^Lokasi harus di Indonesia/);
    });

    it.each(["", "   ", "abc", "Infinity"])(
      "asks for a coordinate instead of 'outside Indonesia' for %j",
      (text) => {
        expect(messages({ ...valid(), lat: text }, "lat")).toEqual([
          "Isi lintang, atau ketuk peta.",
        ]);
        expect(messages({ ...valid(), lng: text }, "lng")).toEqual([
          "Isi bujur, atau ketuk peta.",
        ]);
      },
    );

    it("reports one message for a latitude beyond the globe", () => {
      expect(messages({ ...valid(), lat: "95" }, "lat")).toHaveLength(1);
    });

    it("catches swapped latitude and longitude", () => {
      const values = { ...valid(), lat: "106.8", lng: "-6.2" };
      expect(messages(values, "lat")).not.toEqual([]);
      expect(messages(values, "lng")).not.toEqual([]);
    });
  });
});
