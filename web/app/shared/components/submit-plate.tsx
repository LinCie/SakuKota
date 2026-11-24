import type { ReactNode } from "react";

import { Button } from "./ui/button";

/**
 * The full-width primary button that submits a form. `form` targets a form
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
    <Button
      type="submit"
      form={form}
      disabled={pending}
      className="mt-1 w-full"
    >
      {pending ? "Memproses…" : children}
    </Button>
  );
}
