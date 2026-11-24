import { describe, expect, it } from "vitest";

import {
  formatPrice,
  priceError,
  priceToRequest,
  priceToValues,
} from "./price";

// Intl puts a no-break space after "Rp".
const plain = (text: string | null) => text?.replaceAll("\u00a0", " ") ?? null;

describe("formatPrice", () => {
  it("is null when the price is unknown", () => {
    expect(formatPrice(null, null)).toBeNull();
  });

  it.each([
    [{ min: 9000, max: 12000 }, null, "Rp 9.000–Rp 12.000"],
    [{ min: 9000, max: 12000 }, "kg", "Rp 9.000–Rp 12.000/kg"],
    [{ min: 3500, max: 3500 }, null, "Rp 3.500"],
    [{ min: 0, max: 0 }, null, "Gratis"],
    [{ min: 0, max: 50000 }, null, "Gratis–Rp 50.000"],
    [{ min: 150000, max: 5000000 }, null, "Rp 150.000–Rp 5.000.000"],
  ])("writes %j per %j as %j", (range, unit, expected) => {
    expect(plain(formatPrice(range, unit))).toBe(expected);
  });
});

describe("priceToValues", () => {
  it("is empty when unknown", () => {
    expect(priceToValues(null)).toEqual({ priceMin: "", priceMax: "" });
  });

  it("keeps zero", () => {
    expect(priceToValues({ min: 0, max: 0 })).toEqual({
      priceMin: "0",
      priceMax: "0",
    });
  });
});

describe("priceToRequest", () => {
  it.each([
    ["9000", "12000", { min: 9000, max: 12000 }],
    ["3500", "", { min: 3500, max: 3500 }],
    ["", "3500", { min: 3500, max: 3500 }],
    ["0", "", { min: 0, max: 0 }],
    [" 1500 ", " ", { min: 1500, max: 1500 }],
    ["", "", null],
    ["  ", "\t", null],
  ])("sends %j and %j as %j", (min, max, expected) => {
    expect(priceToRequest(min, max, true)).toEqual(expected);
  });

  it("sends nothing for a category without prices", () => {
    expect(priceToRequest("9000", "12000", false)).toBeNull();
  });
});

describe("priceError", () => {
  it.each([
    ["", ""],
    ["0", ""],
    ["9000", "12000"],
    ["5", "5"],
    ["1000000000", "1000000000"],
  ])("accepts %j and %j", (min, max) => {
    expect(priceError(min, max)).toBeUndefined();
  });

  it.each([
    ["abc", "", "priceMin"],
    ["", "1,5", "priceMax"],
    ["-1", "", "priceMin"],
    ["1e3", "", "priceMin"],
    ["15.000", "", "priceMin"],
    ["", "Rp15000", "priceMax"],
    ["1000000001", "", "priceMin"],
    ["", "1000000001", "priceMax"],
    ["9", "1", "priceMin"],
  ])("rejects %j and %j at %s", (min, max, field) => {
    expect(priceError(min, max)?.field).toBe(field);
  });
});
