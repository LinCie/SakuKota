import { describe, expect, it } from "vitest";

import { field } from "~/features/places/fixtures";
import { toPlaceRequest } from "~/features/places/mappers";

import { suggestionToPlaceValues, toSuggestionRequest } from "./mappers";
import type { Suggestion } from "./types";
import { emptySuggestionValues } from "./values";

const fields = [
  field({ key: "count", data_type: "number" }),
  field({ key: "open", data_type: "boolean" }),
  field({ key: "price", data_type: "range" }),
];

function suggestion(overrides: Partial<Suggestion> = {}): Suggestion {
  return {
    id: 1,
    sub_district_id: 7,
    category_id: 3,
    name: "Warung Bu Tini",
    address: "Jl. Mawar 1",
    location: { type: "Point", coordinates: [106.8, -6.2] },
    operating_hours: "08:00-17:00",
    estimated_price_range: "10-20k",
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
    const request = toSuggestionRequest({ ...values, note: "Dekat pasar" }, []);
    expect(request).toEqual({
      ...toPlaceRequest(values, []),
      note: "Dekat pasar",
    });
  });

  it("sends a blank note as null", () => {
    expect(
      toSuggestionRequest({ ...values, note: "  \n" }, []).note,
    ).toBeNull();
  });

  it("trims the note", () => {
    expect(toSuggestionRequest({ ...values, note: " hi " }, []).note).toBe(
      "hi",
    );
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
      operatingHours: "08:00-17:00",
      priceRange: "10-20k",
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
      operatingHours: "",
      priceRange: "",
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
    const request = toPlaceRequest(suggestionToPlaceValues(s, fields), fields);
    expect(request).toMatchObject({
      sub_district_id: 7,
      category_id: 3,
      name: s.name,
      location: s.location,
      attributes: { count: 0, open: false },
    });
  });
});
