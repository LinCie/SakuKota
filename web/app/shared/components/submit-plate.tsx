import type { ReactNode } from "react";

import { cn } from "~/shared/cn";

import { signButtonClass } from "./menu-board";

/**
 * The full-width yellow sign plate that submits a form. `form` targets a form
 * elsewhere in the page, such as one in a scrolling body above a sticky footer.
 */
export function SubmitPlate({
  pending,
  form,
  children,
}: {
  pending: boolean;
  form?: string;
  children: ReactNode;
}) {
  return (
    <button
      type="submit"
      form={form}
      disabled={pending}
      className={cn(signButtonClass, "disabled:bg-sign-soft mt-1 h-11 w-full")}
    >
      {pending ? "Memproses…" : children}
    </button>
  );
}
