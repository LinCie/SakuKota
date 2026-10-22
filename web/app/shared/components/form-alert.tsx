import { CircleAlert } from "lucide-react";
import type { ReactNode } from "react";

/** A form-level error message. */
export function FormAlert({ children }: { children: ReactNode }) {
  return (
    <div
      role="alert"
      className="border-chili text-ink flex items-start gap-2 rounded-[3px] border-2 px-3 py-2.5 text-sm font-semibold"
    >
      <CircleAlert
        aria-hidden="true"
        className="text-chili mt-0.5 size-4 shrink-0"
        strokeWidth={2.5}
      />
      <p>{children}</p>
    </div>
  );
}
