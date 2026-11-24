import { describe, expect, it } from "vitest";

import { parseCsv, toCsv } from "./csv";

describe("parseCsv", () => {
  it("reads comma-separated records with CRLF or LF line breaks", () => {
    expect(parseCsv("a,b\r\n1,2\n3,4\n")).toEqual([
      ["a", "b"],
      ["1", "2"],
      ["3", "4"],
    ]);
  });

  it("reads quoted cells holding delimiters, quotes, and line breaks", () => {
    expect(parseCsv('a,b\n"x, y","say ""hi""\nthere"')).toEqual([
      ["a", "b"],
      ["x, y", 'say "hi"\nthere'],
    ]);
  });

  it("keeps empty cells", () => {
    expect(parseCsv("a,,c\n,,\n")).toEqual([
      ["a", "", "c"],
      ["", "", ""],
    ]);
  });

  it("detects semicolons when the first line has no comma", () => {
    expect(parseCsv("a;b\n1,5;2\n")).toEqual([
      ["a", "b"],
      ["1,5", "2"],
    ]);
  });

  it("skips a byte order mark", () => {
    expect(parseCsv("\uFEFFa,b\n")).toEqual([["a", "b"]]);
  });

  it("reads nothing from empty text", () => {
    expect(parseCsv("")).toEqual([]);
  });
});

describe("toCsv", () => {
  it("quotes only cells that need it, and round-trips through parseCsv", () => {
    const records = [
      ["name", "note"],
      ["Warung", 'a, "b"; c\nd'],
    ];
    const text = toCsv(records);

    expect(text).toBe('name,note\r\nWarung,"a, ""b""; c\nd"\r\n');
    expect(parseCsv(text)).toEqual(records);
  });
});
