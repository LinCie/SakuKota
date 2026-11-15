import type { Category, CategoryField, Place } from "./types";

// Builders for tests.

export function field(overrides: Partial<CategoryField> = {}): CategoryField {
  return {
    id: 1,
    category_id: 1,
    key: "f",
    label: "F",
    data_type: "text",
    required: false,
    options: null,
    sort_order: 1,
    ...overrides,
  };
}

export const category: Category = {
  id: 3,
  name: "Warung Makan",
  slug: "warung-makan",
  code: "M",
  label: "Makan",
  color: "chili",
};

export function place(overrides: Partial<Place> = {}): Place {
  return {
    id: 1,
    sub_district_id: 7,
    name: "Warung Bu Tini",
    address: "Jl. Mawar 1",
    location: { type: "Point", coordinates: [106.8, -6.2] },
    operating_hours: "08:00-17:00",
    estimated_price_range: "10-20k",
    short_description: "Enak",
    attributes: {},
    rating_average: null,
    rating_count: 0,
    medias: [],
    category,
    ...overrides,
  };
}
