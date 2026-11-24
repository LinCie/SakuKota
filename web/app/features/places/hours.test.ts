import { describe, expect, it } from "vitest";

import {
  emptyHours,
  formatHours,
  hoursError,
  hoursToRequest,
  hoursToValues,
  parseHoursText,
  type HoursValues,
} from "./hours";
import type { OperatingPeriod } from "./types";

const week = (days: number[], open: string, close: string): OperatingPeriod[] =>
  days.map((day) => ({ day, open, close }));

/** Hours values with the given days (1 is Monday) set. */
function values(set: Record<number, HoursValues[number]>): HoursValues {
  return emptyHours().map((day, i) => set[i + 1] ?? day);
}

describe("hoursToValues", () => {
  it("is closed every day without hours", () => {
    expect(hoursToValues(null)).toEqual(emptyHours());
    expect(hoursToValues([])).toEqual(emptyHours());
  });

  it("reads a whole day as its own mode, and a late close as 00:00", () => {
    const result = hoursToValues([
      { day: 1, open: "00:00", close: "24:00" },
      { day: 2, open: "05:00", close: "24:00" },
      { day: 3, open: "08:00", close: "12:00" },
      { day: 3, open: "16:00", close: "20:00" },
    ]);
    expect(result[0]).toEqual({ mode: "24h", periods: [] });
    expect(result[1]).toEqual({
      mode: "hours",
      periods: [{ open: "05:00", close: "00:00" }],
    });
    expect(result[2].periods).toHaveLength(2);
    expect(result[3].mode).toBe("closed");
  });
});

describe("hoursToRequest", () => {
  it("is null when every day is closed", () => {
    expect(hoursToRequest(emptyHours())).toBeNull();
  });

  it("sends 24h days, filled periods, and midnight as 24:00", () => {
    expect(
      hoursToRequest(
        values({
          1: { mode: "24h", periods: [] },
          2: {
            mode: "hours",
            periods: [
              { open: "05:00", close: "00:00" },
              { open: "", close: "" },
            ],
          },
        }),
      ),
    ).toEqual([
      { day: 1, open: "00:00", close: "24:00" },
      { day: 2, open: "05:00", close: "24:00" },
    ]);
  });

  it("round-trips what the backend sends", () => {
    const hours = [
      ...week([1, 2, 3, 4, 5], "08:00", "15:00"),
      { day: 6, open: "08:00", close: "12:00" },
      { day: 6, open: "13:00", close: "24:00" },
      { day: 7, open: "00:00", close: "24:00" },
    ];
    expect(hoursToRequest(hoursToValues(hours))).toEqual(hours);
  });
});

describe("hoursError", () => {
  const monday = (periods: { open: string; close: string }[]) =>
    values({ 1: { mode: "hours", periods } });

  it("accepts closed days, 24 hours, and valid periods", () => {
    expect(hoursError(emptyHours())).toBeUndefined();
    expect(
      hoursError(
        values({
          1: { mode: "24h", periods: [] },
          2: {
            mode: "hours",
            periods: [
              { open: "08:00", close: "12:00" },
              { open: "12:00", close: "00:00" },
            ],
          },
        }),
      ),
    ).toBeUndefined();
  });

  it.each([
    ["no period", [], "Senin: tambahkan jam buka, atau pilih Tutup."],
    [
      "an unfilled time",
      [{ open: "08:00", close: "" }],
      "Senin: isi jam buka dan jam tutup.",
    ],
    [
      "an open after its close",
      [{ open: "22:00", close: "02:00" }],
      "Senin: jam buka harus sebelum jam tutup. Untuk buka lewat tengah malam, isi dua hari.",
    ],
    [
      "equal times",
      [{ open: "08:00", close: "08:00" }],
      "Senin: jam buka harus sebelum jam tutup. Untuk buka lewat tengah malam, isi dua hari.",
    ],
    [
      "overlapping periods",
      [
        { open: "16:00", close: "20:00" },
        { open: "08:00", close: "17:00" },
      ],
      "Senin: jam buka saling tumpang tindih.",
    ],
    [
      "four periods",
      [
        { open: "01:00", close: "02:00" },
        { open: "03:00", close: "04:00" },
        { open: "05:00", close: "06:00" },
        { open: "07:00", close: "08:00" },
      ],
      "Senin: maksimal 3 jam buka per hari.",
    ],
  ])("rejects %s", (_, periods, message) => {
    expect(hoursError(monday(periods))).toBe(message);
  });
});

describe("formatHours", () => {
  it("is empty when the hours are unknown", () => {
    expect(formatHours(null)).toEqual([]);
    expect(formatHours([])).toEqual([]);
  });

  it("names a run of days and the closed day", () => {
    expect(formatHours(week([1, 2, 3, 4, 5, 6], "08:00", "21:00"))).toEqual([
      "Sen–Sab, 08:00–21:00",
      "Min, tutup",
    ]);
  });

  it("says every day, and 24 jam", () => {
    expect(formatHours(week([1, 2, 3, 4, 5, 6, 7], "06:00", "22:00"))).toEqual([
      "Setiap hari, 06:00–22:00",
    ]);
    expect(formatHours(week([1, 2, 3, 4, 5, 6, 7], "00:00", "24:00"))).toEqual([
      "Setiap hari, 24 jam",
    ]);
  });

  it("splits runs where the periods change", () => {
    expect(
      formatHours([
        ...week([1, 2, 3, 4, 5], "08:00", "15:00"),
        ...week([6], "08:00", "12:00"),
        ...week([7], "00:00", "24:00"),
      ]),
    ).toEqual(["Sen–Jum, 08:00–15:00", "Sab, 08:00–12:00", "Min, 24 jam"]);
  });

  it("lists several periods of a day in time order", () => {
    expect(
      formatHours([
        { day: 1, open: "16:00", close: "20:00" },
        { day: 1, open: "08:00", close: "12:00" },
      ]),
    ).toEqual(["Sen, 08:00–12:00, 16:00–20:00", "Sel–Min, tutup"]);
  });
});

describe("parseHoursText", () => {
  it("reads blank text as unknown hours", () => {
    expect(parseHoursText("")).toEqual({ ok: true, hours: null });
    expect(parseHoursText("  ")).toEqual({ ok: true, hours: null });
  });

  it.each([
    ["Setiap hari, 24 jam", week([1, 2, 3, 4, 5, 6, 7], "00:00", "24:00")],
    ["setiap hari 06.00-22.00", week([1, 2, 3, 4, 5, 6, 7], "06:00", "22:00")],
    ["Sen-Jum, 8:00-15:00", week([1, 2, 3, 4, 5], "08:00", "15:00")],
    ["Sen–Sab, 08:00–21:00", week([1, 2, 3, 4, 5, 6], "08:00", "21:00")],
    ["Min, 08:00-13:00", week([7], "08:00", "13:00")],
    ["Setiap hari, 05:00-00:00", week([1, 2, 3, 4, 5, 6, 7], "05:00", "24:00")],
    [
      "Sen-Jum, 08:00-12:00, 17:00-21:00",
      [1, 2, 3, 4, 5].flatMap((day) => [
        { day, open: "08:00", close: "12:00" },
        { day, open: "17:00", close: "21:00" },
      ]),
    ],
    [
      "Sen-Jum, 08:00-20:00; Sab, 08:00-14:00; Min, tutup",
      [
        ...week([1, 2, 3, 4, 5], "08:00", "20:00"),
        ...week([6], "08:00", "14:00"),
      ],
    ],
  ])("reads %j", (text, expected) => {
    const result = parseHoursText(text);
    expect(result.ok && result.hours).toEqual(expected);
  });

  it("reads only closed days as unknown hours", () => {
    expect(parseHoursText("Setiap hari, tutup")).toEqual({
      ok: true,
      hours: null,
    });
  });

  it.each([
    "kapan saja",
    "Sen",
    "Sen-Jum, buka",
    "Sen-Jum, 08:00",
    "Foo-Jum, 08:00-12:00",
    "Jum-Sen, 08:00-12:00",
    "Setiap hari-Min, 08:00-12:00",
  ])("rejects %j", (text) => {
    expect(parseHoursText(text).ok).toBe(false);
  });

  it("reads what formatHours writes", () => {
    const hours = [
      ...week([1, 2, 3, 4, 5], "08:00", "15:00"),
      { day: 6, open: "08:00", close: "12:00" },
      { day: 6, open: "13:00", close: "24:00" },
    ];
    const text = formatHours(hours).join("; ");
    expect(parseHoursText(text)).toEqual({ ok: true, hours });
  });
});
