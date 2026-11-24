import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

// Detail rows for places, and the titled sections of the kecamatan info.

/** A detail row for short values: label left, value right. Rows sit in a `dl` that draws the hairlines between them. */
export function DetailRow({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-baseline justify-between gap-4 py-2.5">
      <dt className="text-muted-foreground max-w-[50%] shrink-0 text-sm font-medium">
        {label}
      </dt>
      <dd className="text-foreground max-w-[65%] text-right font-semibold tabular-nums">
        {children}
      </dd>
    </div>
  );
}

/** A detail block for long values: label above, value below. */
export function DetailBlock({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1 py-2.5">
      <dt className="text-muted-foreground text-sm font-medium">{label}</dt>
      <dd className="text-foreground text-pretty">{children}</dd>
    </div>
  );
}

/** A titled section with an icon; sections are divided by hairlines. */
export function InfoSection({
  icon: Icon,
  title,
  children,
}: {
  icon: LucideIcon;
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-2.5 border-t py-5 first:border-t-0 first:pt-0 last:pb-0">
      <h3 className="text-foreground flex items-center gap-2 text-base font-semibold">
        <Icon aria-hidden={true} className="text-muted-foreground size-5" />
        {title}
      </h3>
      {children}
    </section>
  );
}
