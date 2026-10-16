import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

// Shared look of the menu board: painted rows and dashed dividers.

/** Class names for the board's icon-and-text buttons on paper. */
export const menuButtonClass =
  "text-ink hover:bg-sign-soft focus-visible:bg-sign-soft focus-visible:outline-cat transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2";

/** A menu-board row: label, dotted leader, value. */
export function LeaderRow({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-baseline gap-2 py-2">
      <dt className="text-ink-soft max-w-[50%] shrink-0 text-sm font-semibold">
        {label}
      </dt>
      <span
        aria-hidden="true"
        className="border-ink/25 min-w-6 flex-1 -translate-y-1 border-b-2 border-dotted"
      />
      <dd className="text-ink max-w-[65%] text-right font-semibold tabular-nums">
        {children}
      </dd>
    </div>
  );
}

/** A menu-board block for long values: label above, value below. */
export function StackedRow({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="border-ink/25 flex flex-col gap-1 border-t border-dashed py-2">
      <dt className="text-ink-soft text-sm font-semibold">{label}</dt>
      <dd className="text-ink text-pretty">{children}</dd>
    </div>
  );
}

/** A titled menu-board section with an icon, separated by a dashed rule. */
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
    <section className="border-ink/25 flex flex-col gap-2 border-t border-dashed pt-4 first:border-t-0 first:pt-0">
      <h3 className="text-ink flex items-center gap-2 text-sm font-bold">
        <Icon
          aria-hidden={true}
          className="text-board size-4"
          strokeWidth={2.5}
        />
        {title}
      </h3>
      {children}
    </section>
  );
}
