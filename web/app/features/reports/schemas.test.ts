import { describe, expect, it } from "vitest";

import { reportSchema } from "./schemas";

function problems(values: unknown) {
  const result = reportSchema.safeParse(values);
  if (result.success) return [];
  return result.error.issues.map((issue) => issue.path.join("."));
}

describe("reporting content", () => {
  it("accepts every reason with no note", () => {
    for (const reason of ["spam", "offensive", "fake", "other"]) {
      expect(problems({ reason, note: "" })).toEqual([]);
    }
  });

  it("requires a known reason", () => {
    expect(problems({ reason: "", note: "" })).toEqual(["reason"]);
    expect(problems({ reason: "rude", note: "" })).toEqual(["reason"]);
  });

  it("allows a note of 1000 characters, counted by code point, and rejects 1001", () => {
    expect(problems({ reason: "other", note: "é".repeat(1000) })).toEqual([]);
    expect(problems({ reason: "other", note: "😀".repeat(1000) })).toEqual([]);
    expect(problems({ reason: "other", note: "a".repeat(1001) })).toEqual([
      "note",
    ]);
  });
});
