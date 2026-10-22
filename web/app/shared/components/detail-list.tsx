import type { ReactNode } from "react";

import { cn } from "~/shared/cn";

export type DetailItem = {
  label: string;
  value: ReactNode;
  /** Spans both columns, for long text. */
  wide?: boolean;
};

/** Label and value pairs laid out in two columns on wide screens. */
export function DetailList({ items }: { items: DetailItem[] }) {
  return (
    <dl className="grid gap-x-8 gap-y-5 sm:grid-cols-2">
      {items.map((item) => (
        <div
          key={item.label}
          className={cn(
            "flex min-w-0 flex-col gap-1",
            item.wide && "sm:col-span-2",
          )}
        >
          <dt className="text-ink-soft text-xs font-semibold">{item.label}</dt>
          <dd className="text-ink text-sm text-pretty break-words">
            {item.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}
