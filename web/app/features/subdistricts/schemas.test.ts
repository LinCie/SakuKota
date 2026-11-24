import { describe, expect, it } from "vitest";

import {
  emptySubDistrictValues,
  toSubDistrictRequest,
  type SubDistrictValues,
} from "./mappers";
import { subDistrictSchema } from "./schemas";

const valid: SubDistrictValues = {
  ...emptySubDistrictValues,
  name: "Menteng",
  slug: "menteng",
};

describe("subDistrictSchema", () => {
  it("accepts a name and slug alone", () => {
    expect(subDistrictSchema.safeParse(valid).success).toBe(true);
  });

  it.each([
    ["a blank name", { name: "  " }],
    ["a long name", { name: "a".repeat(101) }],
    ["a blank slug", { slug: "" }],
    ["a long slug", { slug: "a".repeat(101) }],
  ])("rejects %s", (_, change) => {
    expect(subDistrictSchema.safeParse({ ...valid, ...change }).success).toBe(
      false,
    );
  });

  it.each(["0", "75000", "75000,5", "75000.25", "99999999.99"])(
    "accepts spending %s",
    (dailySpending) => {
      expect(
        subDistrictSchema.safeParse({ ...valid, dailySpending }).success,
      ).toBe(true);
    },
  );

  it.each(["50.000", "-1", "1e3", "abc", "100000000"])(
    "rejects spending %s",
    (dailySpending) => {
      expect(
        subDistrictSchema.safeParse({ ...valid, dailySpending }).success,
      ).toBe(false);
    },
  );
});

describe("toSubDistrictRequest", () => {
  it("sends blanks as null", () => {
    expect(toSubDistrictRequest({ ...valid, floodHistory: " " })).toEqual({
      name: "Menteng",
      slug: "menteng",
      flood_history: null,
      safety_crime_information: null,
      estimated_daily_spending: null,
    });
  });

  it("reads a decimal comma", () => {
    expect(
      toSubDistrictRequest({ ...valid, dailySpending: "75000,5" })
        .estimated_daily_spending,
    ).toBe(75000.5);
  });
});
