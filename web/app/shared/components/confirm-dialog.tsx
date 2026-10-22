import { AlertDialog } from "@base-ui/react/alert-dialog";
import type { ReactNode } from "react";

import { Button } from "./button";

type ConfirmDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: ReactNode;
  confirmLabel: string;
  /** Destructive confirmations read as danger; the rest as primary. */
  tone?: "danger" | "primary";
  pending: boolean;
  error: string | null;
  onConfirm: () => void;
};

/** A modal that asks the user to confirm an action before it runs. */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  tone = "danger",
  pending,
  error,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <AlertDialog.Root
      open={open}
      onOpenChange={(next) => {
        // Closing mid-request would hide the outcome.
        if (!pending) onOpenChange(next);
      }}
    >
      <AlertDialog.Portal>
        <AlertDialog.Backdrop className="bg-ink/40 fixed inset-0 z-50 transition-opacity duration-150 data-ending-style:opacity-0 data-starting-style:opacity-0" />
        <AlertDialog.Viewport className="fixed inset-0 z-50 grid place-items-center p-4">
          <AlertDialog.Popup className="border-ink/15 bg-paint text-ink flex w-full max-w-sm flex-col gap-4 rounded-[5px] border p-5 shadow-[0_24px_60px_rgb(0_0_0/0.35)] transition-[opacity,translate] duration-150 outline-none data-ending-style:translate-y-2 data-ending-style:opacity-0 data-starting-style:translate-y-2 data-starting-style:opacity-0">
            <div className="flex flex-col gap-1.5">
              <AlertDialog.Title className="text-base font-bold">
                {title}
              </AlertDialog.Title>
              <AlertDialog.Description className="text-ink-soft text-sm text-pretty">
                {description}
              </AlertDialog.Description>
            </div>
            {error ? (
              <p role="alert" className="text-chili text-sm font-semibold">
                {error}
              </p>
            ) : null}
            <div className="flex justify-end gap-2">
              <AlertDialog.Close
                render={<Button variant="secondary" disabled={pending} />}
              >
                Batal
              </AlertDialog.Close>
              <Button variant={tone} disabled={pending} onClick={onConfirm}>
                {pending ? "Memproses…" : confirmLabel}
              </Button>
            </div>
          </AlertDialog.Popup>
        </AlertDialog.Viewport>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
