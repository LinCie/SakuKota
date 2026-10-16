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
