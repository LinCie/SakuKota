// Indonesian formatting for user-facing values.

const numberFormat = new Intl.NumberFormat("id-ID");
const rupiahFormat = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});
const dateFormat = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "long",
  timeZone: "UTC",
});
const localDateFormat = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "long",
});
const localMonthFormat = new Intl.DateTimeFormat("id-ID", {
  month: "long",
  year: "numeric",
});
const ratingFormat = new Intl.NumberFormat("id-ID", {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
});

export function formatNumber(value: number): string {
  return numberFormat.format(value);
}

export function formatRupiah(value: number): string {
  return rupiahFormat.format(value);
}

/** Formats in UTC, so a `YYYY-MM-DD` date parsed as UTC midnight keeps its day. */
export function formatDate(date: Date): string {
  return dateFormat.format(date);
}

/** Formats a moment, such as a `created_at` timestamp, as its day where the viewer is. */
export function formatLocalDate(date: Date): string {
  return localDateFormat.format(date);
}

/** Formats a moment's month where the viewer is, such as "November 2026". */
export function formatLocalMonth(date: Date): string {
  return localMonthFormat.format(date);
}

/** Formats a star rating with one decimal, such as 4,3. */
export function formatRating(value: number): string {
  return ratingFormat.format(value);
}
