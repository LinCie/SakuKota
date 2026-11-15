import { describe, expect, it } from "vitest";

import { emptySuggestionValues } from "./values";
import { rejectSchema, suggestionSchema } from "./schemas";

const valid = () => ({
  ...emptySuggestionValues([]),
  subDistrictId: "7",
  categoryId: "3",
  name: "Warung Bu Tini",
  lat: "-6.2",
  lng: "106.8",
});

function problems(
  schema: typeof suggestionSchema | typeof rejectSchema,
  values: unknown,
) {
  const result = schema.safeParse(values);
  if (result.success) return [];
  return result.error.issues.map((issue) => issue.path.join("."));
}

describe("suggesting a place", () => {
  it("accepts a place with no note", () => {
    expect(problems(suggestionSchema, valid())).toEqual([]);
  });

  it("holds the suggestion to the place's own rules", () => {
    expect(
      problems(suggestionSchema, { ...valid(), name: "  ", lat: "40" }).sort(),
    ).toEqual(["lat", "name"]);
  });

  it("allows a note of 1000 characters and rejects 1001", () => {
    expect(
      problems(suggestionSchema, { ...valid(), note: "a".repeat(1000) }),
    ).toEqual([]);
    expect(
      problems(suggestionSchema, { ...valid(), note: "a".repeat(1001) }),
    ).toEqual(["note"]);
  });

  it("counts a note by characters, not UTF-16 units", () => {
    expect(
      problems(suggestionSchema, { ...valid(), note: "😀".repeat(1000) }),
    ).toEqual([]);
  });
});

describe("rejecting a suggestion", () => {
  it("does not require a reason", () => {
    expect(problems(rejectSchema, { reason: "" })).toEqual([]);
  });

  it("allows 1000 characters and rejects 1001, ignoring surrounding spaces", () => {
    expect(problems(rejectSchema, { reason: ` ${"a".repeat(1000)} ` })).toEqual(
      [],
    );
    expect(problems(rejectSchema, { reason: "a".repeat(1001) })).toEqual([
      "reason",
    ]);
  });
});
