import { describe, expect, it } from "vitest";

import { field } from "./fixtures";
import {
  emptyPlaceValues,
  isBlank,
  parseNumber,
  rangeOf,
  textOf,
} from "./values";

describe("parseNumber", () => {
  it.each(["", "   ", "abc", "-", ".", "1,5,3", "1.000,5", "1 5"])(
    "rejects %j",
    (text) => {
      expect(parseNumber(text)).toBeNaN();
    },
  );

  it.each([
    ["1,5", 1.5],
    ["1.5", 1.5],
    ["  -6,2  ", -6.2],
    ["0", 0],
    ["-0,5", -0.5],
  ])("reads %j as %d", (text, expected) => {
    expect(parseNumber(text)).toBe(expected);
  });

  // Number() accepts these, but nobody types them into a price or coordinate.
  it.each(["Infinity", "-Infinity", "1e3", "0x10", "0b11"])(
    "does not accept %j as a number",
    (text) => {
      expect(Number.isFinite(parseNumber(text))).toBe(false);
    },
  );

  it("rejects a number too large to be finite", () => {
    expect(parseNumber("9".repeat(400))).toBeNaN();
  });
});

describe("isBlank", () => {
  it("treats empty and whitespace text as blank", () => {
    expect(isBlank("", field())).toBe(true);
    expect(isBlank("   ", field())).toBe(true);
    expect(isBlank(undefined, field())).toBe(true);
  });

  it("does not treat '0' or 'false' as blank", () => {
    expect(isBlank("0", field({ data_type: "number" }))).toBe(false);
    expect(isBlank("false", field({ data_type: "boolean" }))).toBe(false);
  });

  it("treats a range as blank only when both ends are empty", () => {
    const range = field({ data_type: "range" });
    expect(isBlank({ min: "", max: "" }, range)).toBe(true);
    expect(isBlank({ min: " ", max: " " }, range)).toBe(true);
    expect(isBlank(undefined, range)).toBe(true);
    expect(isBlank({ min: "1", max: "" }, range)).toBe(false);
    expect(isBlank({ min: "", max: "1" }, range)).toBe(false);
  });

  it("treats a range field holding text as blank", () => {
    expect(isBlank("5", field({ data_type: "range" }))).toBe(true);
  });
});

describe("textOf / rangeOf", () => {
  it("fall back to empty for a value of the other shape", () => {
    expect(textOf({ min: "1", max: "2" })).toBe("");
    expect(textOf(undefined)).toBe("");
    expect(rangeOf("5")).toEqual({ min: "", max: "" });
    expect(rangeOf(undefined)).toEqual({ min: "", max: "" });
  });
});

describe("emptyPlaceValues", () => {
  const subDistrict = (id: number) => ({ id }) as never;

  it("preselects a lone kecamatan only", () => {
    expect(emptyPlaceValues([subDistrict(4)]).subDistrictId).toBe("4");
    expect(emptyPlaceValues([]).subDistrictId).toBe("");
    expect(
      emptyPlaceValues([subDistrict(4), subDistrict(5)]).subDistrictId,
    ).toBe("");
  });

  it("does not share the attributes object between calls", () => {
    const a = emptyPlaceValues([]);
    const b = emptyPlaceValues([]);
    a.attributes.x = "1";
    expect(b.attributes).toEqual({});
  });
});
