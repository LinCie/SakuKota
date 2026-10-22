import { Toast } from "@base-ui/react/toast";
import type { ReactNode } from "react";

import { toastManager } from "~/shared/toast-manager";

import { ToastList } from "./toast-list";

/** Provides the toast queue to `children` and draws it bottom-right. */
export function Toaster({ children }: { children: ReactNode }) {
  return (
    <Toast.Provider toastManager={toastManager}>
      {children}
      <Toast.Portal>
        <Toast.Viewport className="fixed right-4 bottom-4 z-60 flex w-[min(100%-2rem,22rem)] flex-col gap-2 outline-none">
          <ToastList />
        </Toast.Viewport>
      </Toast.Portal>
    </Toast.Provider>
  );
}
