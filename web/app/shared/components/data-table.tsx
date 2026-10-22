import type { ReactNode } from "react";

import { cn } from "~/shared/cn";

export type Column<T> = {
  key: string;
  header: string;
  cell: (row: T) => ReactNode;
  className?: string;
  /** Keeps the column in view while the table scrolls sideways on a phone. */
  pinned?: boolean;
};

type DataTableProps<T> = {
  columns: Column<T>[];
  rows: T[];
  getRowKey: (row: T) => string | number;
  /** Shown in place of the rows when there are none. */
  empty: ReactNode;
};

/** A dense table with hairline rows; the last column holds row actions. */
export function DataTable<T>({
  columns,
  rows,
  getRowKey,
  empty,
}: DataTableProps<T>) {
  return (
    <div className="border-ink/15 bg-paint overflow-x-auto rounded-[5px] border">
      <table className="w-full min-w-max border-collapse text-left text-sm">
        <thead className="bg-paint-deep text-ink-soft text-xs font-semibold">
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                scope="col"
                className={cn(
                  "h-10 px-4 font-semibold whitespace-nowrap",
                  column.pinned && "bg-paint-deep sticky left-0",
                  column.className,
                )}
              >
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={getRowKey(row)}
              className="border-ink/10 hover:bg-paint-deep/50 border-t transition-colors"
            >
              {columns.map((column) => (
                <td
                  key={column.key}
                  className={cn(
                    "px-4 py-3 align-middle",
                    column.pinned && "bg-paint sticky left-0",
                    column.className,
                  )}
                >
                  {column.cell(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      {rows.length === 0 ? (
        <div className="p-6 text-center">{empty}</div>
      ) : null}
    </div>
  );
}
