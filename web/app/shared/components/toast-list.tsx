import { Toast } from "@base-ui/react/toast";
import { X } from "lucide-react";

import { Button } from "~/shared/components/ui/button";

/** The queued toasts, each with a title, optional description, and close. */
export function ToastList() {
  const { toasts } = Toast.useToastManager();

  return toasts.map((toast) => (
    <Toast.Root
      key={toast.id}
      toast={toast}
      className="bg-card text-foreground shadow-float flex items-start gap-3 rounded-2xl p-4 text-sm transition-[translate,opacity] duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] data-ending-style:opacity-0 data-starting-style:translate-y-2 data-starting-style:opacity-0"
    >
      <Toast.Content className="flex min-w-0 flex-1 flex-col gap-0.5">
        <Toast.Title className="font-semibold" />
        {toast.description ? (
          <Toast.Description className="text-muted-foreground text-pretty" />
        ) : null}
      </Toast.Content>
      <Toast.Close
        aria-label="Tutup notifikasi"
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground -my-1.5 -mr-1.5"
          />
        }
      >
        <X aria-hidden="true" />
      </Toast.Close>
    </Toast.Root>
  ));
}
