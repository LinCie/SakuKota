import { createFileRoute, redirect } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { AdminShell } from "~/features/admin/components/admin-shell";
import { getSession } from "~/features/auth/session";

export const Route = createFileRoute("/admin")({
  beforeLoad: ({ location }) => {
    if (!getSession()) {
      throw redirect({
        to: "/masuk",
        search: { kembali: location.href },
        replace: true,
      });
    }
  },
  head: () => ({ meta: [{ title: "Admin · Rantau" }] }),
  component: AdminShell,
  errorComponent: AdminRouteError,
});
