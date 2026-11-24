import type { ReactNode } from "react";

/** A titled group of fields inside a form. */
export function FormSection({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-4">
      <h3 className="text-muted-foreground text-xs font-semibold tracking-wide uppercase">
        {title}
      </h3>
      {children}
    </section>
  );
}
