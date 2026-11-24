import { Drawer } from "@base-ui/react/drawer";
import { X } from "lucide-react";
import { useState, type ReactNode } from "react";

import { Button } from "./ui/button";
import { ConfirmDialog } from "./confirm-dialog";

type FormDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Called once the closing animation has finished; reset the form here. */
  onClosed: () => void;
  title: string;
  description: string;
  /** Unsaved input: closing asks first, and a tap outside is ignored. */
  dirty: boolean;
  /** The `id` of the `<form>` in `children`; the footer's submit button targets it. */
  formId: string;
  pending: boolean;
  submitLabel: string;
  children: ReactNode;
};

/** A right-side drawer that holds a form, with a fixed header and footer. */
export function FormDrawer({
  open,
  onOpenChange,
  onClosed,
  title,
  description,
  dirty,
  formId,
  pending,
  submitLabel,
  children,
}: FormDrawerProps) {
  const [confirming, setConfirming] = useState(false);

  return (
    <Drawer.Root
      open={open}
      swipeDirection="right"
      onOpenChange={(next, details) => {
        if (next) return;
        if (pending || dirty) {
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
      <Drawer.Portal>
        <Drawer.Backdrop className="bg-foreground/30 fixed inset-0 z-40 transition-opacity duration-200 data-ending-style:opacity-0 data-starting-style:opacity-0" />
        <Drawer.Viewport className="fixed inset-0 z-50 flex justify-end">
          <Drawer.Popup className="bg-card text-foreground shadow-dialog flex h-full w-full max-w-lg flex-col transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] outline-none data-ending-style:translate-x-full data-starting-style:translate-x-full">
            <header className="flex items-start justify-between gap-4 border-b px-5 py-4">
              <div className="flex min-w-0 flex-col gap-1">
                <Drawer.Title className="text-lg font-semibold">
                  {title}
                </Drawer.Title>
                <Drawer.Description className="text-muted-foreground text-sm text-pretty">
                  {description}
                </Drawer.Description>
              </div>
              <Drawer.Close
                aria-label="Tutup"
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="text-muted-foreground -mt-1 -mr-2"
                  />
                }
              >
                <X aria-hidden="true" />
              </Drawer.Close>
            </header>

            {/* Content: a mouse press on the form never starts the swipe, which
                would capture the pointer and swallow the click. */}
            <Drawer.Content className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 py-5">
              {children}
            </Drawer.Content>

            <footer className="flex justify-end gap-2 border-t px-5 py-3">
              <Drawer.Close
                render={
                  <Button variant="outline" size="sm" disabled={pending} />
                }
              >
                Batal
              </Drawer.Close>
              <Button type="submit" form={formId} size="sm" disabled={pending}>
                {pending ? "Menyimpan…" : submitLabel}
              </Button>
            </footer>

            <ConfirmDialog
              open={confirming}
              onOpenChange={setConfirming}
              title="Buang isian?"
              description="Yang sudah kamu isi belum disimpan dan akan hilang."
              confirmLabel="Buang isian"
              pending={false}
              error={null}
              onConfirm={() => {
                setConfirming(false);
                onOpenChange(false);
              }}
            />
          </Drawer.Popup>
        </Drawer.Viewport>
      </Drawer.Portal>
    </Drawer.Root>
  );
}
