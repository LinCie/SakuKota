import type { ReactNode } from "react";

import { cn } from "~/shared/cn";
import { Wordmark } from "./wordmark";

/**
 * The top bar of the map, `/usulan`, and `/profil`: the wordmark, then the
 * page's own controls. By default they share one row, the last control at the
 * far end; `className` replaces that layout.
 */
export function SiteHeader({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <header className="bg-background border-b">
      <div
        className={cn(
          "flex items-center justify-between gap-3 px-3 py-2 md:px-5 md:py-3",
          className,
        )}
      >
        <Wordmark />
        {children}
      </div>
    </header>
  );
}
