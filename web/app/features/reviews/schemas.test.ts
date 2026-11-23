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
    expect(problems({ rating: 4, comment: "", files: [] })).toEqual({});
  });

  it("asks to pick stars when none is picked", () => {
    expect(problems({ rating: 0, comment: "Enak", files: [] })).toEqual({
      rating: ["Pilih jumlah bintang."],
    });
  });

  it.each([1, 5])("accepts %d stars", (rating) => {
    expect(problems({ rating, comment: "", files: [] })).toEqual({});
  });

  it.each([6, -1, 2.5, Number.NaN])("rejects %d stars", (rating) => {
    expect(problems({ rating, comment: "", files: [] })).toHaveProperty(
      "rating",
    );
  });

  describe("comment", () => {
    it("allows 1000 characters and rejects 1001", () => {
      expect(
        problems({ rating: 5, comment: "a".repeat(1000), files: [] }),
      ).toEqual({});
      expect(
        problems({ rating: 5, comment: "a".repeat(1001), files: [] }),
      ).toEqual({
        comment: ["Komentar maksimal 1000 karakter."],
      });
    });

    it("does not count surrounding whitespace", () => {
      expect(
        problems({ rating: 5, comment: `  ${"a".repeat(1000)}\n`, files: [] }),
      ).toEqual({});
    });

    // The backend counts characters, so a emoji is one, not two.
    it("counts an emoji as one character", () => {
      expect(
        problems({ rating: 5, comment: "😀".repeat(1000), files: [] }),
      ).toEqual({});
      expect(
        problems({ rating: 5, comment: "😀".repeat(1001), files: [] }),
      ).toHaveProperty("comment");
    });
  });

  describe("files", () => {
    const file = () => new File([], "foto.png", { type: "image/png" });

    it("allows 10 files and rejects 11", () => {
      const ten = Array.from({ length: 10 }, file);
      expect(problems({ rating: 5, comment: "", files: ten })).toEqual({});
      expect(
        problems({ rating: 5, comment: "", files: [...ten, file()] }),
      ).toEqual({ files: ["Maksimal 10 file."] });
    });
  });
});
