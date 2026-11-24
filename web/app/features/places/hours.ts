import type { OperatingPeriod } from "./types";

// A place's operating hours: the form's values, the request's periods, how
// they read to people, and the text a CSV import takes. The backend's rules
// (domain.NormalizeOperatingHours) are mirrored by `hoursError`.

/** The most periods one day may have, as the backend allows. */
export const MAX_PERIODS_PER_DAY = 3;

/** The weekdays, Monday first, as the backend numbers them. */
export const DAYS = [
  { day: 1, short: "Sen", long: "Senin" },
  { day: 2, short: "Sel", long: "Selasa" },
  { day: 3, short: "Rab", long: "Rabu" },
  { day: 4, short: "Kam", long: "Kamis" },
  { day: 5, short: "Jum", long: "Jumat" },
  { day: 6, short: "Sab", long: "Sabtu" },
  { day: 7, short: "Min", long: "Minggu" },
] as const;

const MINUTES_PER_DAY = 24 * 60;

// Form values

/** A period as typed into two time inputs: "HH:MM", or "" while unfilled. */
export type PeriodValues = { open: string; close: string };

/**
 * One day as typed. A time input cannot hold 24:00, so a day that is open all
 * day is its own mode, and a close of 00:00 means midnight.
 */
export type DayValues = {
  mode: "closed" | "24h" | "hours";
  periods: PeriodValues[];
};

/** The seven days, Monday first. */
export type HoursValues = DayValues[];

export function emptyHours(): HoursValues {
  return DAYS.map(() => ({ mode: "closed", periods: [] }));
}

/** An API time as a time input holds it: 24:00 is typed as 00:00. */
function toInputTime(time: string): string {
  return time === "24:00" ? "00:00" : time;
}

/** The place's periods as form values. */
export function hoursToValues(hours: OperatingPeriod[] | null): HoursValues {
  return DAYS.map(({ day }): DayValues => {
    const periods = (hours ?? []).filter((p) => p.day === day);
    if (periods.length === 0) return { mode: "closed", periods: [] };
    if (
      periods.length === 1 &&
      periods[0].open === "00:00" &&
      periods[0].close === "24:00"
    ) {
      return { mode: "24h", periods: [] };
    }
    return {
      mode: "hours",
      periods: periods.map((p) => ({
        open: p.open,
        close: toInputTime(p.close),
      })),
    };
  });
}

/** The request's periods: filled periods only, or null when no day is open. */
export function hoursToRequest(values: HoursValues): OperatingPeriod[] | null {
  const periods: OperatingPeriod[] = [];

  values.forEach((value, i) => {
    const day = DAYS[i].day;
    if (value.mode === "24h") {
      periods.push({ day, open: "00:00", close: "24:00" });
    } else if (value.mode === "hours") {
      for (const { open, close } of value.periods) {
        if (open === "" || close === "") continue;
        periods.push({ day, open, close: close === "00:00" ? "24:00" : close });
      }
    }
  });

  return periods.length > 0 ? periods : null;
}

/** Minutes since midnight of an HH:MM time, or NaN. 00:00 as a close is midnight. */
function minutesOf(time: string, isClose: boolean): number {
  const match = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(time);
  if (!match) return NaN;
  const minutes = Number(match[1]) * 60 + Number(match[2]);
  return isClose && minutes === 0 ? MINUTES_PER_DAY : minutes;
}

/** The first problem in the hours, naming its day, or undefined when they are valid. */
export function hoursError(values: HoursValues): string | undefined {
  for (const [i, value] of values.entries()) {
    if (value.mode !== "hours") continue;
    const { long } = DAYS[i];

    if (value.periods.length === 0) {
      return `${long}: tambahkan jam buka, atau pilih Tutup.`;
    }
    if (value.periods.length > MAX_PERIODS_PER_DAY) {
      return `${long}: maksimal ${MAX_PERIODS_PER_DAY} jam buka per hari.`;
    }

    const spans: [number, number][] = [];
    for (const { open, close } of value.periods) {
      const from = minutesOf(open, false);
      const to = minutesOf(close, true);
      if (Number.isNaN(from) || Number.isNaN(to)) {
        return `${long}: isi jam buka dan jam tutup.`;
      }
      if (from >= to) {
        return `${long}: jam buka harus sebelum jam tutup. Untuk buka lewat tengah malam, isi dua hari.`;
      }
      spans.push([from, to]);
    }

    spans.sort((a, b) => a[0] - b[0]);
    for (let j = 1; j < spans.length; j++) {
      if (spans[j][0] < spans[j - 1][1]) {
        return `${long}: jam buka saling tumpang tindih.`;
      }
    }
  }
  return undefined;
}

// Reading

function clock(period: OperatingPeriod): string {
  return `${period.open}–${period.close}`;
}

/**
 * The hours as lines for people, one per run of days that share their periods:
 * "Sen–Sab, 08:00–21:00", "Min, tutup", "Setiap hari, 24 jam". Empty when the
 * hours are unknown.
 */
export function formatHours(hours: OperatingPeriod[] | null): string[] {
  if (!hours || hours.length === 0) return [];

  const signatures = DAYS.map(({ day }) => {
    const periods = hours
      .filter((p) => p.day === day)
      .sort((a, b) => a.open.localeCompare(b.open));
    if (periods.length === 0) return "tutup";
    if (
      periods.length === 1 &&
      periods[0].open === "00:00" &&
      periods[0].close === "24:00"
    ) {
      return "24 jam";
    }
    return periods.map(clock).join(", ");
  });

  const lines: string[] = [];
  let start = 0;
  for (let i = 1; i <= DAYS.length; i++) {
    if (i < DAYS.length && signatures[i] === signatures[start]) continue;

    const days =
      i - start === DAYS.length
        ? "Setiap hari"
        : i - start === 1
          ? DAYS[start].short
          : `${DAYS[start].short}–${DAYS[i - 1].short}`;
    lines.push(`${days}, ${signatures[start]}`);
    start = i;
  }

  return lines;
}

// CSV text

/** The periods a CSV cell holds, or what is wrong with it. */
export type HoursTextResult =
  { ok: true; hours: OperatingPeriod[] | null } | { ok: false; error: string };

const dayNumbers = new Map<string, number>(
  DAYS.map(({ day, short }) => [short.toLowerCase(), day]),
);

const syntax = `Contoh: "Sen-Jum, 08:00-20:00; Sab, 08:00-14:00", "Setiap hari, 24 jam".`;

/**
 * Reads hours written as `formatHours` writes them, with a plain hyphen or an
 * en dash and one or more periods per day: "Sen-Jum, 08:00-20:00; Sab, 09:00-13:00,
 * 16:00-20:00; Min, tutup". A blank text is unknown hours. Days left out are closed.
 * Only the form is read here; `hoursError` judges the periods.
 */
export function parseHoursText(text: string): HoursTextResult {
  const trimmed = text.trim();
  if (trimmed === "") return { ok: true, hours: null };

  const hours: OperatingPeriod[] = [];
  for (const segment of trimmed.split(";")) {
    const match =
      /^\s*(setiap hari|[a-z]{3})(?:\s*[-–]\s*([a-z]{3}))?\s*,?\s*(.+?)\s*$/i.exec(
        segment,
      );
    if (!match)
      return { ok: false, error: `jam buka tidak terbaca. ${syntax}` };

    const [, first, last, rest] = match;
    let days: number[];
    if (first.toLowerCase() === "setiap hari") {
      if (last)
        return { ok: false, error: `jam buka tidak terbaca. ${syntax}` };
      days = DAYS.map(({ day }) => day);
    } else {
      const from = dayNumbers.get(first.toLowerCase());
      const to = last ? dayNumbers.get(last.toLowerCase()) : from;
      if (from === undefined || to === undefined || from > to) {
        return {
          ok: false,
          error: `hari "${segment.trim()}" tidak dikenal. Gunakan Sen, Sel, Rab, Kam, Jum, Sab, atau Min.`,
        };
      }
      days = DAYS.map(({ day }) => day).filter((d) => d >= from && d <= to);
    }

    let spans: [string, string][];
    if (/^24\s*jam$/i.test(rest)) {
      spans = [["00:00", "24:00"]];
    } else if (/^tutup$/i.test(rest)) {
      spans = [];
    } else {
      spans = [];
      for (const part of rest.split(",")) {
        const time =
          /^\s*(\d{1,2})[:.](\d{2})\s*[-–]\s*(\d{1,2})[:.](\d{2})\s*$/.exec(
            part,
          );
        if (!time) {
          return {
            ok: false,
            error: `jam "${part.trim()}" tidak terbaca. ${syntax}`,
          };
        }
        const pad = (n: string) => n.padStart(2, "0");
        const close = `${pad(time[3])}:${time[4]}`;
        spans.push([
          `${pad(time[1])}:${time[2]}`,
          close === "00:00" ? "24:00" : close,
        ]);
      }
    }

    for (const day of days) {
      for (const [open, close] of spans) hours.push({ day, open, close });
    }
  }

  return { ok: true, hours: hours.length > 0 ? hours : null };
}
