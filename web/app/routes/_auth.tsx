import { createFileRoute, redirect } from "@tanstack/react-router";

import { AuthShell } from "~/features/auth/components/auth-shell";
import { getSession } from "~/features/auth/session";

type AuthSearch = {
  /** Where to go after signing in: the page the visitor came from. */
  kembali?: string;
};

// Only paths inside the app are accepted (never another origin, never the
// auth pages themselves); anything else returns to the map.
function returnPath(value: unknown) {
  if (typeof value !== "string" || !value.startsWith("/")) return undefined;
  if (value.startsWith("//") || value.startsWith("/\\")) return undefined;
  if (/^\/(masuk|daftar)(?:[/?#]|$)/.test(value)) return undefined;
  return value;
}

export const Route = createFileRoute("/_auth")({
  validateSearch: (search: Record<string, unknown>): AuthSearch => ({
    kembali: returnPath(search.kembali),
  }),
  beforeLoad: ({ search }) => {
    if (getSession()) {
      throw redirect({ href: search.kembali ?? "/", replace: true });
    }
  },
  component: AuthRoute,
});

function AuthRoute() {
  const { kembali } = Route.useSearch();
  return <AuthShell kembali={kembali} />;
}
