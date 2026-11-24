import type { ReactNode } from "react";

/**
 * A page's title block: an optional way back above it, the title with the
 * page's actions beside it, and the description under both. When the actions
 * do not fit beside the title, they drop under it.
 */
export function PageHeader({
  title,
  description,
  actions,
  back,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
  /** A link back to the page this one sits under. */
  back?: ReactNode;
}) {
  return (
    <header className="flex flex-col gap-1">
      {back ? <div className="mb-1 flex">{back}</div> : null}
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
        <h1 className="min-w-0 flex-[1_1_12rem] text-2xl leading-9 font-semibold tracking-[-0.01em] text-balance">
          {title}
        </h1>
        {actions ? (
          <div className="flex flex-wrap items-center gap-2">{actions}</div>
        ) : null}
      </div>
      {description ? (
        <p className="text-muted-foreground max-w-prose text-sm text-pretty">
          {description}
        </p>
      ) : null}
    </header>
  );
}
