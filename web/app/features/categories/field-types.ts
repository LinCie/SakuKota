/** The data types a category field can take, with their labels in the admin. */
const FIELD_TYPES = [
  { value: "text", label: "Teks" },
  { value: "number", label: "Angka" },
  { value: "boolean", label: "Ya / tidak" },
  { value: "select", label: "Pilihan" },
  { value: "range", label: "Rentang angka" },
  { value: "date", label: "Tanggal" },
] as const;

export function fieldTypeLabel(dataType: string): string {
  return FIELD_TYPES.find((type) => type.value === dataType)?.label ?? dataType;
}
