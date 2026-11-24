// CSV (RFC 4180) reading and writing for spreadsheet imports and templates.

/**
 * The records of a CSV text, each a list of cells. Quoted cells may hold the
 * delimiter, line breaks, and doubled quotes. The delimiter is a comma, or a
 * semicolon when the first line has one and no comma, as spreadsheets set to
 * a decimal-comma locale save. A leading byte order mark is skipped.
 */
export function parseCsv(text: string): string[][] {
  const source = text.startsWith("\uFEFF") ? text.slice(1) : text;
  const firstLine = source.split(/\r?\n/, 1)[0];
  const delimiter =
    firstLine.includes(";") && !firstLine.includes(",") ? ";" : ",";

  const records: string[][] = [];
  let record: string[] = [];
  let cell = "";
  let quoted = false;

  for (let i = 0; i < source.length; i++) {
    const char = source[i];

    if (quoted) {
      if (char !== '"') cell += char;
      else if (source[i + 1] === '"') {
        cell += '"';
        i++;
      } else quoted = false;
      continue;
    }

    if (char === '"') quoted = true;
    else if (char === delimiter) {
      record.push(cell);
      cell = "";
    } else if (char === "\n" || char === "\r") {
      // \r\n ends one record, not two.
      if (char === "\r" && source[i + 1] === "\n") i++;
      record.push(cell);
      records.push(record);
      record = [];
      cell = "";
    } else cell += char;
  }

  // The last record may lack a line break.
  if (cell !== "" || record.length > 0) {
    record.push(cell);
    records.push(record);
  }

  return records;
}

/** CSV text of the records, with CRLF line breaks; cells are quoted only when needed. */
export function toCsv(records: string[][]): string {
  return records
    .map((record) =>
      record
        .map((cell) =>
          /[",\r\n;]/.test(cell) ? `"${cell.replaceAll('"', '""')}"` : cell,
        )
        .join(","),
    )
    .join("\r\n")
    .concat("\r\n");
}

/** Saves the records as a CSV file through the browser's download. */
export function downloadCsv(filename: string, records: string[][]) {
  // The byte order mark makes spreadsheets read the file as UTF-8.
  const blob = new Blob(["\uFEFF", toCsv(records)], {
    type: "text/csv;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}
