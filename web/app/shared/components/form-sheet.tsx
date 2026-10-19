import { AlertDialog } from "@base-ui/react/alert-dialog";
import { Dialog } from "@base-ui/react/dialog";
import { X } from "lucide-react";
import { useRef, useState, type ReactNode } from "react";

import { cn } from "~/shared/cn";

import { SubmitPlate } from "./form-fields";
import { signButtonClass } from "./menu-board";

type FormSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Called once the closing animation has finished; reset the form here. */
  onClosed: () => void;
  title: string;
  /** Unsaved input: closing asks first, and taps outside are ignored. */
  dirty: boolean;
  /** The `id` of the `<form>` in `children`; the footer plate submits it. */
  formId: string;
  pending: boolean;
  submitLabel: string;
  children: ReactNode;
};

const plateButton =
  "font-sign text-ink focus-visible:outline-cat h-11 rounded-[3px] border-2 px-4 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2";

/**
 * A form in a Base UI dialog: a bottom sheet on phones, a centred notice board
 * from 768px. The header and the submit plate stay fixed while the body scrolls.
 */
export function FormSheet({
  open,
  onOpenChange,
  onClosed,
  title,
  dirty,
  formId,
  pending,
  submitLabel,
  children,
}: FormSheetProps) {
  const [confirming, setConfirming] = useState(false);
  const closeRef = useRef<HTMLButtonElement>(null);

  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next, details) => {
        if (next) return;
        if (dirty || pending) {
          details.cancel();
          // A stray tap outside must not throw the input away or nag.
          if (!pending && details.reason !== "outside-press") {
            setConfirming(true);
          }
          return;
        }
        onOpenChange(false);
      }}
      onOpenChangeComplete={(isOpen) => {
        if (!isOpen) onClosed();
      }}
    >
      <Dialog.Portal>
        <Dialog.Backdrop className="bg-ink/60 fixed inset-0 z-1200 data-ending-style:opacity-0 data-starting-style:opacity-0 motion-safe:transition-opacity motion-safe:duration-300" />
        <Dialog.Viewport className="fixed inset-0 z-1200 flex items-end justify-center md:items-center md:p-3">
          <Dialog.Popup
            initialFocus={closeRef}
            className="border-ink bg-paint text-ink flex max-h-[92dvh] w-full flex-col rounded-t-[10px] border-2 border-b-0 shadow-[0_-10px_30px_rgb(26_26_23/0.3)] outline-none data-ending-style:translate-y-full data-starting-style:translate-y-full motion-safe:transition-[translate,opacity] motion-safe:duration-300 motion-safe:ease-out md:max-h-[calc(100dvh-1.5rem)] md:max-w-xl md:rounded-md md:border-b-2 md:shadow-[0_24px_60px_rgb(0_0_0/0.35)] md:data-ending-style:translate-y-4 md:data-ending-style:opacity-0 md:data-starting-style:translate-y-4 md:data-starting-style:opacity-0"
          >
            <header className="bg-board text-paint shrink-0 rounded-t-[7px] p-1.5 md:rounded-t-[3px]">
              <div className="border-sign flex items-center justify-between gap-3 rounded-[3px] border-2 py-2 pr-2 pl-4">
                <Dialog.Title className="font-sign text-sign text-xl leading-tight">
                  {title}
                </Dialog.Title>
                <Dialog.Close
                  ref={closeRef}
                  aria-label="Tutup"
                  className="text-paint hover:bg-board-deep focus-visible:bg-board-deep focus-visible:outline-sign grid size-9 shrink-0 place-items-center rounded-[3px] transition-colors focus-visible:outline-2"
                >
                  <X aria-hidden="true" className="size-5" strokeWidth={2.5} />
                </Dialog.Close>
              </div>
            </header>
            <div aria-hidden="true" className="bg-chili h-1.5 shrink-0" />

            <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-4 sm:p-5">
              {children}
            </div>

            <footer className="border-ink bg-paint shrink-0 border-t-2 p-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] sm:px-5">
              <SubmitPlate form={formId} pending={pending}>
                {submitLabel}
              </SubmitPlate>
            </footer>

            <AlertDialog.Root open={confirming} onOpenChange={setConfirming}>
              <AlertDialog.Portal>
                <AlertDialog.Backdrop className="bg-ink/60 fixed inset-0 z-1300 data-ending-style:opacity-0 data-starting-style:opacity-0 motion-safe:transition-opacity motion-safe:duration-200" />
                <AlertDialog.Viewport className="fixed inset-0 z-1300 grid place-items-center p-3">
                  <AlertDialog.Popup className="border-ink bg-paint text-ink flex w-full max-w-sm flex-col gap-4 rounded-[5px] border-2 p-5 shadow-[0_24px_60px_rgb(0_0_0/0.35)] data-ending-style:translate-y-2 data-ending-style:opacity-0 data-starting-style:translate-y-2 data-starting-style:opacity-0 motion-safe:transition-[translate,opacity] motion-safe:duration-200 motion-safe:ease-out">
                    <div className="flex flex-col gap-1.5">
                      <AlertDialog.Title className="font-sign text-lg leading-tight">
                        Buang isian?
                      </AlertDialog.Title>
                      <AlertDialog.Description className="text-ink-soft text-pretty">
                        Yang sudah kamu isi belum disimpan dan akan hilang.
                      </AlertDialog.Description>
                    </div>
                    <div className="flex flex-col gap-2 sm:flex-row-reverse">
                      <AlertDialog.Close
                        className={cn(signButtonClass, "h-11 sm:flex-1")}
                      >
                        Lanjut mengisi
                      </AlertDialog.Close>
                      <button
                        type="button"
                        onClick={() => {
                          setConfirming(false);
                          onOpenChange(false);
                        }}
                        className={`${plateButton} border-ink/60 bg-paint hover:border-ink border-dashed hover:border-solid sm:flex-1`}
                      >
                        Buang isian
                      </button>
                    </div>
                  </AlertDialog.Popup>
                </AlertDialog.Viewport>
              </AlertDialog.Portal>
            </AlertDialog.Root>
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
