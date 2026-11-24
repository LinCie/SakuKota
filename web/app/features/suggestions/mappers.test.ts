import { describe, expect, it } from "vitest";

import { field } from "~/features/places/fixtures";
import { emptyHours } from "~/features/places/hours";
import { toPlaceRequest } from "~/features/places/mappers";

import {
  emptySuggestionValues,
  suggestionToPlaceValues,
  toSuggestionRequest,
} from "./mappers";
import type { Suggestion } from "./types";

const fields = [
  field({ key: "count", data_type: "number" }),
  field({ key: "open", data_type: "boolean" }),
  field({ key: "price", data_type: "range" }),
];

const category = { fields, has_price: true };

function suggestion(overrides: Partial<Suggestion> = {}): Suggestion {
  return {
    id: 1,
    sub_district_id: 7,
    category_id: 3,
    name: "Warung Bu Tini",
    address: "Jl. Mawar 1",
    location: { type: "Point", coordinates: [106.8, -6.2] },
    operating_hours: [{ day: 1, open: "08:00", close: "17:00" }],
    estimated_price_range: { min: 10000, max: 20000 },
    short_description: "Enak",
    attributes: {},
    note: null,
    status: "pending",
    rejection_reason: null,
    place_id: null,
    reviewed_at: null,
    created_at: "2025-01-01T00:00:00Z",
    user: { id: 2, name: "Siti" },
    ...overrides,
  };
}

describe("sending a suggestion", () => {
  const values = {
    ...emptySuggestionValues([]),
    subDistrictId: "7",
    categoryId: "3",
    name: "Warung",
    lat: "-6.2",
    lng: "106.8",
  };

  it("is the place request plus the note", () => {
    const request = toSuggestionRequest(
      { ...values, note: "Dekat pasar" },
      category,
    );
    expect(request).toEqual({
      ...toPlaceRequest(values, category),
      note: "Dekat pasar",
    });
  });

  it("sends a blank note as null", () => {
    expect(
      toSuggestionRequest({ ...values, note: "  \n" }, category).note,
    ).toBeNull();
  });

  it("trims the note", () => {
    expect(
      toSuggestionRequest({ ...values, note: " hi " }, category).note,
    ).toBe("hi");
  });
});

describe("approving a suggestion", () => {
  it("starts the place form from what the user suggested", () => {
    const values = suggestionToPlaceValues(
      suggestion({
        attributes: { count: 0, open: false, price: { min: 1, max: 2 } },
      }),
      fields,
    );
    expect(values).toEqual({
      subDistrictId: "7",
      categoryId: "3",
      name: "Warung Bu Tini",
      lat: "-6.2",
      lng: "106.8",
      address: "Jl. Mawar 1",
      hours: expect.arrayContaining([
        { mode: "hours", periods: [{ open: "08:00", close: "17:00" }] },
      ]),
      priceMin: "10000",
      priceMax: "20000",
      description: "Enak",
      attributes: {
        count: "0",
        open: "false",
        price: { min: "1", max: "2" },
      },
    });
  });

  it("copes with a suggestion that left everything optional empty", () => {
    const values = suggestionToPlaceValues(
      suggestion({
        address: null,
        location: null,
        operating_hours: null,
        estimated_price_range: null,
        short_description: null,
      }),
      fields,
    );
    expect(values).toMatchObject({
      lat: "",
      lng: "",
      address: "",
      hours: emptyHours(),
      priceMin: "",
      priceMax: "",
      description: "",
    });
  });

  it("leaves out answers for fields the category no longer has", () => {
    expect(
      suggestionToPlaceValues(suggestion({ attributes: { gone: "x" } }), fields)
        .attributes,
    ).toEqual({});
  });

  it("comes out as the same place when sent unchanged", () => {
    const s = suggestion({ attributes: { count: 0, open: false } });
    const request = toPlaceRequest(
      suggestionToPlaceValues(s, fields),
      category,
    );
    expect(request).toMatchObject({
      sub_district_id: 7,
      category_id: 3,
      name: s.name,
      location: s.location,
      operating_hours: s.operating_hours,
      estimated_price_range: s.estimated_price_range,
      attributes: { count: 0, open: false },
    });
  });
});
