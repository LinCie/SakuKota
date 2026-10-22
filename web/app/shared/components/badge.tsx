import type { ReactNode } from "react";

import { cn } from "~/shared/cn";

/** A small inline label with a hairline border. */
export function Badge({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-[3px] border px-2 py-0.5 text-xs font-semibold",
        className,
      )}
    >
      {children}
    </span>
  );
}
