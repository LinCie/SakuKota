import { describe, expect, it } from "vitest";

import { category, field, place } from "./fixtures";
import { attributesToValues, placeToValues, toPlaceRequest } from "./mappers";
import { attributeErrors } from "./schemas";
import { emptyPlaceValues, type PlaceValues } from "./values";

const fields = [
  field({ key: "text", data_type: "text" }),
  field({ key: "count", data_type: "number" }),
  field({ key: "open", data_type: "boolean" }),
  field({ key: "price", data_type: "range" }),
  field({ key: "since", data_type: "date" }),
  field({ key: "kind", data_type: "select" }),
];

describe("placeToValues", () => {
  it("reads GeoJSON [lng, lat] into lat and lng", () => {
    const values = placeToValues(
      place({ location: { type: "Point", coordinates: [106.8, -6.2] } }),
      [],
    );
    expect(values.lat).toBe("-6.2");
    expect(values.lng).toBe("106.8");
  });

  it("leaves coordinates empty when the place has no location", () => {
    const values = placeToValues(place({ location: null }), []);
    expect(values.lat).toBe("");
    expect(values.lng).toBe("");
  });

  it("turns null text fields into empty strings, never 'null'", () => {
    const values = placeToValues(
      place({
        address: null,
        operating_hours: null,
        estimated_price_range: null,
        short_description: null,
      }),
      [],
    );
    expect(values).toMatchObject({
      address: "",
      operatingHours: "",
      priceRange: "",
      description: "",
    });
  });

  it("takes ids as strings", () => {
    const values = placeToValues(place({ sub_district_id: 7 }), []);
    expect(values.subDistrictId).toBe("7");
    expect(values.categoryId).toBe(String(category.id));
  });
});

describe("attributesToValues", () => {
  it("keeps zero and false", () => {
    const values = attributesToValues(fields, {
      count: 0,
      open: false,
      price: { min: 0, max: 0 },
    });
    expect(values).toEqual({
      count: "0",
      open: "false",
      price: { min: "0", max: "0" },
    });
  });

  it("cuts a date timestamp to YYYY-MM-DD", () => {
    expect(
      attributesToValues(fields, { since: "2025-03-04T00:00:00Z" }).since,
    ).toBe("2025-03-04");
  });

  it("skips values of the wrong type instead of showing them", () => {
    const values = attributesToValues(fields, {
      open: "true", // a string for a boolean
      price: { min: 1 }, // half a range
      since: 20250304, // a number for a date
      count: { nested: true }, // an object for a number
      text: ["a"], // an array for text
    });
    expect(values).toEqual({});
  });

  it("skips null and missing values", () => {
    expect(
      attributesToValues(fields, { count: null, open: null, price: null }),
    ).toEqual({});
  });

  it("ignores attributes the category has no field for", () => {
    expect(attributesToValues(fields, { gone: "x" })).toEqual({});
  });

  it("reads a field of an unknown type as text", () => {
    expect(
      attributesToValues([field({ key: "c", data_type: "color" })], {
        c: "red",
      }),
    ).toEqual({ c: "red" });
  });
});

describe("toPlaceRequest", () => {
  const values = (overrides: Partial<PlaceValues> = {}): PlaceValues => ({
    ...emptyPlaceValues([]),
    subDistrictId: "7",
    categoryId: "3",
    name: "Warung",
    lat: "-6.2",
    lng: "106.8",
    ...overrides,
  });

  it("writes coordinates as [lng, lat]", () => {
    expect(toPlaceRequest(values(), []).location).toEqual({
      type: "Point",
      coordinates: [106.8, -6.2],
    });
  });

  it("reads decimal commas in coordinates", () => {
    expect(
      toPlaceRequest(values({ lat: "-6,2", lng: "106,8" }), []).location
        ?.coordinates,
    ).toEqual([106.8, -6.2]);
  });

  it("trims the name and sends blank optional text as null", () => {
    const request = toPlaceRequest(
      values({
        name: "  Warung  ",
        address: "   ",
        operatingHours: "",
        priceRange: "\t",
        description: " ",
      }),
      [],
    );
    expect(request).toMatchObject({
      name: "Warung",
      address: null,
      operating_hours: null,
      estimated_price_range: null,
      short_description: null,
    });
  });

  it("sends ids as numbers", () => {
    expect(toPlaceRequest(values(), [])).toMatchObject({
      sub_district_id: 7,
      category_id: 3,
    });
  });

  describe("attributes", () => {
    const send = (attributes: PlaceValues["attributes"]) =>
      toPlaceRequest(values({ attributes }), fields).attributes;

    it("omits blank fields", () => {
      expect(
        send({
          text: "  ",
          count: "",
          open: "",
          price: { min: "", max: "" },
          since: "",
        }),
      ).toEqual({});
    });

    it("keeps zero and false", () => {
      expect(
        send({ count: "0", open: "false", price: { min: "0", max: "0" } }),
      ).toEqual({ count: 0, open: false, price: { min: 0, max: 0 } });
    });

    it("types each value by its field", () => {
      expect(
        send({
          text: " halal ",
          count: "1,5",
          open: "true",
          price: { min: "10", max: "20,5" },
          since: "2025-03-04",
          kind: "warung",
        }),
      ).toEqual({
        text: "halal",
        count: 1.5,
        open: true,
        price: { min: 10, max: 20.5 },
        since: "2025-03-04",
        kind: "warung",
      });
    });

    // Switching category in the form leaves the old category's values behind.
    it("drops values of fields the category does not have", () => {
      expect(send({ count: "3", stale: "from the old category" })).toEqual({
        count: 3,
      });
    });

    it("never sends NaN or Infinity for input the form accepts", () => {
      const inputs = [
        "0",
        "1,5",
        "-3",
        "abc",
        "Infinity",
        "1e999",
        "0x10",
        "-",
      ];
      for (const text of inputs) {
        const attributes = { count: text, price: { min: "0", max: text } };
        if (Object.keys(attributeErrors(fields, attributes)).length > 0) {
          continue;
        }
        const sent = JSON.stringify(send(attributes));
        expect(sent, `input ${JSON.stringify(text)}`).not.toContain("null");
      }
    });
  });
});

describe("round trip", () => {
  it("returns the same attributes after loading a place into the form and saving", () => {
    const attributes = {
      text: "halal",
      count: 0,
      open: false,
      price: { min: 0, max: 20 },
      since: "2025-03-04",
      kind: "warung",
    };
    const p = place({ attributes });
    const request = toPlaceRequest(placeToValues(p, fields), fields);
    expect(request.attributes).toEqual(attributes);
  });

  it("returns the same place fields", () => {
    const p = place();
    const request = toPlaceRequest(placeToValues(p, []), []);
    expect(request).toEqual({
      sub_district_id: p.sub_district_id,
      category_id: p.category.id,
      name: p.name,
      address: p.address,
      location: p.location,
      operating_hours: p.operating_hours,
      estimated_price_range: p.estimated_price_range,
      short_description: p.short_description,
      attributes: {},
    });
  });

  it("produces values that pass attribute validation", () => {
    const p = place({
      attributes: { count: 0, open: false, price: { min: 0, max: 20 } },
    });
    expect(
      attributeErrors(fields, placeToValues(p, fields).attributes),
    ).toEqual({});
  });
});
