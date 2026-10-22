import { useMemo, useState } from "react";

/**
 * Filters table rows by a search term, matched case-insensitively against the
 * text `textOf` returns for each row. `textOf` should be a stable function.
 */
export function useTableQuery<T>(rows: T[], textOf: (row: T) => string) {
  const [query, setQuery] = useState("");
  const needle = query.trim().toLowerCase();

  const filtered = useMemo(
    () =>
      needle === ""
        ? rows
        : rows.filter((row) => textOf(row).toLowerCase().includes(needle)),
    [rows, needle, textOf],
  );

  return { query, setQuery, rows: filtered };
}
