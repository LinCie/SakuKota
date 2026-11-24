import { Flag } from "lucide-react";
import { lazy, Suspense, useState } from "react";

import { isAdmin, useSession } from "~/features/auth/session";
import { cn } from "~/shared/cn";
import { buttonVariants } from "~/shared/components/ui/button";

// The drawer pulls in zod and TanStack Form, so it loads on the first click.
const ReportDrawer = lazy(() =>
  import("./report-drawer").then((module) => ({
    default: module.ReportDrawer,
  })),
);

type ReportButtonProps = {
  placeId: number;
  /** The reported review; null reports the place itself. */
  reviewId: number | null;
  /** Names the content mid-sentence, e.g. "ulasan Ana" or "tempat Warung Bu Tini". */
  target: string;
  className?: string;
};

/**
 * "Laporkan" for signed-in users, opening the report drawer. Hidden from
 * signed-out visitors, and from admins, who delete content directly.
 */
export function ReportButton({
  placeId,
  reviewId,
  target,
  className,
}: ReportButtonProps) {
  const session = useSession();
  const [open, setOpen] = useState(false);
  // Mounted from the first click on, so the drawer can animate closed.
  const [opened, setOpened] = useState(false);

  if (!session || isAdmin(session)) return null;

  return (
    <>
      <button
        type="button"
        onClick={() => {
          setOpened(true);
          setOpen(true);
        }}
        aria-label={`Laporkan ${target}`}
        className={cn(
          buttonVariants({ variant: "ghost", size: "sm" }),
          "text-muted-foreground",
          className,
        )}
      >
        <Flag aria-hidden="true" data-icon="inline-start" />
        Laporkan
      </button>
      {opened ? (
        <Suspense fallback={null}>
          <ReportDrawer
            open={open}
            onOpenChange={setOpen}
            placeId={placeId}
            reviewId={reviewId}
            target={target}
          />
        </Suspense>
      ) : null}
    </>
  );
}
