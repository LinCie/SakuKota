import type { ComponentProps } from "react";

import { cn } from "~/shared/cn";

/** A pulsing block standing in for content that is still loading. */
export function Skeleton({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "bg-ink/10 rounded-[3px] motion-safe:animate-pulse",
        className,
      )}
      {...props}
    />
  );
}
