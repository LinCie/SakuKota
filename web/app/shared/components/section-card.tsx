import type { ReactNode } from "react";

/** A soft-white card with a heading, for one group of details. */
export function SectionCard({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="bg-card ring-foreground/10 flex min-w-0 flex-col gap-4 rounded-xl p-5 ring-1">
      <h2 className="text-foreground text-sm font-semibold">{title}</h2>
      {children}
    </section>
  );
}
