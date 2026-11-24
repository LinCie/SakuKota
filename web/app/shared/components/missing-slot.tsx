import type { ReactNode } from "react";

/** A quiet, explicit gap: missing data is shown as missing, never hidden. */
export function MissingSlot({ children }: { children: ReactNode }) {
  return (
    <span className="bg-muted text-muted-foreground inline-block rounded-md px-2 py-0.5 text-sm font-normal">
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
