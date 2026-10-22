/** A text value as a nullable string: blank becomes null. */
export function orNull(text: string) {
  const trimmed = text.trim();
  return trimmed === "" ? null : trimmed;
}

/** A lowercase, hyphen-separated slug, with accents removed. */
export function slugify(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}
