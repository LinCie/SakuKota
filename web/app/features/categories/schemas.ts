import { z } from "zod";

export const categorySchema = z.object({
  name: z.string().trim().min(1, "Nama kategori wajib diisi."),
  slug: z
    .string()
    .trim()
    .min(1, "Slug wajib diisi.")
    .regex(
      /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
      "Slug hanya huruf kecil, angka, dan tanda hubung.",
    ),
});

export type CategoryValues = z.infer<typeof categorySchema>;

/** One option per line; blank lines are dropped. */
export function parseOptions(text: string): string[] {
  return text
    .split("\n")
    .map((option) => option.trim())
    .filter(Boolean);
}

export const categoryFieldSchema = z
  .object({
    label: z.string().trim().min(1, "Label wajib diisi."),
    key: z
      .string()
      .trim()
      .min(1, "Kunci wajib diisi.")
      .regex(
        /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
        "Kunci hanya huruf kecil, angka, dan tanda hubung.",
      ),
    dataType: z.string().min(1, "Pilih tipe data."),
    required: z.boolean(),
    options: z.string(),
  })
  .refine(
    (values) =>
      values.dataType !== "select" || parseOptions(values.options).length > 0,
    { message: "Isi minimal satu pilihan.", path: ["options"] },
  );

export type CategoryFieldValues = z.infer<typeof categoryFieldSchema>;
