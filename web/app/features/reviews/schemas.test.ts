import { describe, expect, it } from "vitest";

import { reviewSchema } from "./schemas";

function problems(values: unknown) {
  const result = reviewSchema.safeParse(values);
  if (result.success) return {};
  const byField: Record<string, string[]> = {};
  for (const issue of result.error.issues) {
    (byField[issue.path.join(".")] ??= []).push(issue.message);
  }
  return byField;
}

describe("writing a review", () => {
  it("accepts a rating alone, without a comment", () => {
    expect(problems({ rating: 4, comment: "" })).toEqual({});
  });

  it("asks to pick stars when none is picked", () => {
    expect(problems({ rating: 0, comment: "Enak" })).toEqual({
      rating: ["Pilih jumlah bintang."],
    });
  });

  it.each([1, 5])("accepts %d stars", (rating) => {
    expect(problems({ rating, comment: "" })).toEqual({});
  });

  it.each([6, -1, 2.5, Number.NaN])("rejects %d stars", (rating) => {
    expect(problems({ rating, comment: "" })).toHaveProperty("rating");
  });

  describe("comment", () => {
    it("allows 1000 characters and rejects 1001", () => {
      expect(problems({ rating: 5, comment: "a".repeat(1000) })).toEqual({});
      expect(problems({ rating: 5, comment: "a".repeat(1001) })).toEqual({
        comment: ["Komentar maksimal 1000 karakter."],
      });
    });

    it("does not count surrounding whitespace", () => {
      expect(
        problems({ rating: 5, comment: `  ${"a".repeat(1000)}\n` }),
      ).toEqual({});
    });

    // The backend counts characters, so a emoji is one, not two.
    it("counts an emoji as one character", () => {
      expect(problems({ rating: 5, comment: "😀".repeat(1000) })).toEqual({});
      expect(
        problems({ rating: 5, comment: "😀".repeat(1001) }),
      ).toHaveProperty("comment");
    });
  });
});
