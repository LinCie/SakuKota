import { Outlet } from "@tanstack/react-router";

import { isAdmin, useSession } from "~/features/auth/session";
import { Toaster } from "~/shared/components/toaster";

import { AdminGate } from "./admin-gate";
import { AdminSidebar } from "./admin-sidebar";

/** The admin console's frame: navigation with the account, and the page; gated to admins. */
export function AdminShell() {
  const session = useSession();
  if (!isAdmin(session)) return <AdminGate signedIn={session !== null} />;

  return (
    <Toaster>
      <div className="bg-background text-foreground flex min-h-dvh flex-col md:grid md:grid-cols-[15rem_minmax(0,1fr)]">
        <AdminSidebar />
        <main className="flex min-w-0 flex-1 flex-col gap-6 p-4 md:p-6">
          <Outlet />
        </main>
      </div>
    </Toaster>
  );
}
