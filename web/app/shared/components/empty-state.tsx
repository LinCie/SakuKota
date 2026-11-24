import type { ReactNode } from "react";

/** A table or page with nothing in it yet, with the action that fills it. */
export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center gap-2">
      <p className="font-semibold">{title}</p>
      {description ? (
        <p className="text-muted-foreground max-w-sm text-sm text-pretty">
          {description}
        </p>
      ) : null}
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  );
}
