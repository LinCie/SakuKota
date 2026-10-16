import type { ReactNode } from "react";

/** A drawn empty slot: missing data is shown as missing, never hidden. */
export function MissingSlot({ children }: { children: ReactNode }) {
  return (
    <span className="border-ink/30 text-ink-soft inline-block rounded-[3px] border-2 border-dashed px-2 py-0.5 text-sm">
      {children}
    </span>
  );
}

/** The value, or a drawn slot naming what is missing when it is null or empty. */
export function orMissing(value: ReactNode, message = "Data tidak tersedia") {
  return value === null || value === undefined || value === "" ? (
    <MissingSlot>{message}</MissingSlot>
  ) : (
    value
  );
}
