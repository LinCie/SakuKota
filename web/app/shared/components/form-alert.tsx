import { CircleAlert } from "lucide-react";
import type { ReactNode } from "react";

/** A form-level error message. */
export function FormAlert({ children }: { children: ReactNode }) {
  return (
    <div
      role="alert"
      className="bg-sambal-soft text-sambal-deep flex items-start gap-2 rounded-xl px-3 py-2.5 text-sm font-medium"
    >
      <CircleAlert aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <p>{children}</p>
    </div>
  );
}
