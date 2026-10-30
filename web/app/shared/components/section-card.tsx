import type { ReactNode } from "react";

/** A bordered panel with a heading, for one group of details. */
export function SectionCard({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="border-ink/15 bg-paint flex min-w-0 flex-col gap-4 rounded-[5px] border p-5">
      <h2 className="text-sm font-bold">{title}</h2>
      {children}
    </section>
  );
}
