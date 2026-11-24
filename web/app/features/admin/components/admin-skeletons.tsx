import type { ReactNode } from "react";

import { Skeleton } from "~/shared/components/skeleton";

/** Announces the loading to assistive tech while the skeleton is drawn. */
function Loading({ children }: { children: ReactNode }) {
  return (
    <>
      <span role="status" className="sr-only">
        Memuat…
      </span>
      {children}
    </>
  );
}

function HeaderSkeleton({ actions }: { actions: number }) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div className="flex min-w-0 flex-col gap-1">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-5 w-72 max-w-full" />
      </div>
      {actions > 0 ? (
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {Array.from({ length: actions }, (_, i) => (
            <Skeleton key={i} className="h-9 w-28" />
          ))}
        </div>
      ) : null}
    </div>
  );
}

const TABLE_ROWS = 8;

/** The shape of an admin list page: header, search and count, and a table. */
export function AdminListSkeleton({
  action = false,
  filter = false,
}: {
  /** The header has a button, as on the places list. */
  action?: boolean;
  /** The toolbar has a select beside the search, as on the suggestions list. */
  filter?: boolean;
}) {
  return (
    <Loading>
      <HeaderSkeleton actions={action ? 1 : 0} />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex w-full max-w-xl flex-wrap items-center gap-3 sm:flex-nowrap">
          <Skeleton className="h-9 w-full max-w-sm" />
          {filter ? <Skeleton className="h-9 w-full sm:w-44" /> : null}
        </div>
        <Skeleton className="h-5 w-28" />
      </div>

      <div className="bg-card ring-foreground/10 overflow-hidden rounded-xl ring-1">
        <div className="bg-muted flex h-10 items-center gap-8 px-4">
          <Skeleton className="bg-foreground/10 h-3 w-24" />
          <Skeleton className="bg-foreground/10 h-3 w-16" />
          <Skeleton className="bg-foreground/10 hidden h-3 w-20 sm:block" />
          <Skeleton className="bg-foreground/10 hidden h-3 w-20 md:block" />
        </div>
        {Array.from({ length: TABLE_ROWS }, (_, i) => (
          <div key={i} className="flex items-center gap-8 border-t px-4 py-3">
            <div className="flex min-w-0 flex-1 flex-col gap-0.5 sm:max-w-56 sm:flex-none sm:basis-56">
              <Skeleton className="h-5 w-40 max-w-full" />
              <Skeleton className="h-4 w-56 max-w-full" />
            </div>
            <Skeleton className="h-5 w-20" />
            <Skeleton className="hidden h-5 w-24 sm:block" />
            <Skeleton className="hidden h-5 w-24 md:block" />
          </div>
        ))}
      </div>
    </Loading>
  );
}

function CardSkeleton({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="bg-card ring-foreground/10 flex min-w-0 flex-col gap-4 rounded-xl p-5 ring-1">
      <h2 className="text-foreground text-sm font-semibold">{title}</h2>
      {children}
    </div>
  );
}

/** The shape of an admin detail page: back link, header, and a grid of cards. */
export function AdminDetailSkeleton({
  actions,
  cards,
}: {
  /** How many buttons the header has. */
  actions: number;
  /** One per card: its title, and a number of detail items or `"image"`. */
  cards: Array<{ title: string; content: number | "image" }>;
}) {
  return (
    <Loading>
      <Skeleton className="h-9 w-36 self-start" />
      <HeaderSkeleton actions={actions} />

      <div className="grid gap-5 lg:grid-cols-2">
        {cards.map(({ title, content }) => (
          <CardSkeleton key={title} title={title}>
            {content === "image" ? (
              <>
                <Skeleton className="aspect-4/3 w-full" />
                <Skeleton className="h-9 w-36" />
              </>
            ) : (
              <div className="grid gap-x-8 gap-y-5 sm:grid-cols-2">
                {Array.from({ length: content }, (_, j) => (
                  <div key={j} className="flex flex-col gap-1">
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-5 w-36 max-w-full" />
                  </div>
                ))}
              </div>
            )}
          </CardSkeleton>
        ))}
      </div>
    </Loading>
  );
}
