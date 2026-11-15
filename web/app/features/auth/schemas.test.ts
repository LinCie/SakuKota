import { describe, expect, it } from "vitest";
import type { z } from "zod";

import { signInSchema, signUpSchema } from "./schemas";

/** The messages a visitor would see, by field. */
function problems(schema: z.ZodType, values: unknown) {
  const result = schema.safeParse(values);
  if (result.success) return {};
  const byField: Record<string, string[]> = {};
  for (const issue of result.error.issues) {
    (byField[issue.path.join(".")] ??= []).push(issue.message);
  }
  return byField;
}

describe("signing in", () => {
  it("accepts an email and a password", () => {
    expect(
      problems(signInSchema, { email: "siti@example.com", password: "x" }),
    ).toEqual({});
  });

  it("asks for a password when it is empty, without judging its length", () => {
    expect(
      problems(signInSchema, { email: "siti@example.com", password: "" }),
    ).toEqual({ password: ["Kata sandi wajib diisi."] });
    // An old account may have a short password; sign-in must let it through.
    expect(
      problems(signInSchema, { email: "siti@example.com", password: "abc" }),
    ).toEqual({});
  });

  it.each(["", "siti", "siti@", "@example.com", "siti example@x.com"])(
    "rejects the email %j",
    (email) => {
      expect(problems(signInSchema, { email, password: "x" })).toEqual({
        email: ["Format email belum benar, contoh: nama@email.com."],
      });
    },
  );

  it("reports both fields when both are wrong", () => {
    expect(
      Object.keys(problems(signInSchema, { email: "", password: "" })),
    ).toEqual(["email", "password"]);
  });
});

describe("signing up", () => {
  const valid = {
    name: "Siti",
    email: "siti@example.com",
    password: "12345678",
  };

  it("accepts a valid account", () => {
    expect(problems(signUpSchema, valid)).toEqual({});
  });

  describe("name", () => {
    it.each(["", "   ", "\t"])("rejects %j", (name) => {
      expect(problems(signUpSchema, { ...valid, name })).toEqual({
        name: ["Nama wajib diisi."],
      });
    });

    it("allows 100 characters and rejects 101", () => {
      expect(
        problems(signUpSchema, { ...valid, name: "a".repeat(100) }),
      ).toEqual({});
      expect(
        problems(signUpSchema, { ...valid, name: "a".repeat(101) }),
      ).toEqual({
        name: ["Nama maksimal 100 karakter."],
      });
    });

    it("does not count surrounding spaces toward the limit", () => {
      expect(
        problems(signUpSchema, { ...valid, name: ` ${"a".repeat(100)} ` }),
      ).toEqual({});
    });
  });

  describe("password", () => {
    it("requires 8 characters", () => {
      expect(problems(signUpSchema, { ...valid, password: "1234567" })).toEqual(
        {
          password: ["Kata sandi minimal 8 karakter."],
        },
      );
      expect(
        problems(signUpSchema, { ...valid, password: "12345678" }),
      ).toEqual({});
    });

    it("does not trim it: spaces are part of a password", () => {
      expect(
        problems(signUpSchema, { ...valid, password: "        " }),
      ).toEqual({});
    });
  });

  it("rejects a malformed email", () => {
    expect(problems(signUpSchema, { ...valid, email: "siti@" })).toEqual({
      email: ["Format email belum benar, contoh: nama@email.com."],
    });
  });
});
