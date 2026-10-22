import { Toast } from "@base-ui/react/toast";
import { X } from "lucide-react";

/** The queued toasts, each with a title, optional description, and close. */
export function ToastList() {
  const { toasts } = Toast.useToastManager();

  return toasts.map((toast) => (
    <Toast.Root
      key={toast.id}
      toast={toast}
      className="border-ink/15 bg-ink text-paint flex items-start gap-3 rounded-[5px] border p-3.5 text-sm shadow-[0_8px_24px_rgb(26_26_23/0.25)] transition-[translate,opacity] duration-200 data-ending-style:opacity-0 data-starting-style:translate-y-2 data-starting-style:opacity-0"
    >
      <Toast.Content className="flex min-w-0 flex-1 flex-col gap-0.5">
        <Toast.Title className="font-semibold" />
        {toast.description ? (
          <Toast.Description className="text-paint/80" />
        ) : null}
      </Toast.Content>
      <Toast.Close
        aria-label="Tutup notifikasi"
        className="text-paint/80 hover:text-paint focus-visible:outline-sign grid size-6 shrink-0 place-items-center rounded-[3px] focus-visible:outline-2"
      >
        <X aria-hidden="true" className="size-4" strokeWidth={2.5} />
      </Toast.Close>
    </Toast.Root>
  ));
}
