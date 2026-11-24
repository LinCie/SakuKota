import { Link } from "@tanstack/react-router";

import { AccountControl } from "~/features/auth/components/account-control";
import { Badge } from "~/shared/components/ui/badge";

/** The admin's top bar: the console label, a way back to the map, and the account. */
export function AdminTopbar() {
  return (
    <header className="bg-background sticky top-0 z-10 flex h-14 items-center justify-between gap-3 border-b px-4 md:px-6">
      <div className="flex min-w-0 items-center gap-4">
        <Badge variant="outline">Admin</Badge>
        <Link
          to="/"
          className="text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 rounded-md text-sm font-medium transition-colors outline-none focus-visible:ring-3"
        >
          Kembali ke peta
        </Link>
      </div>
      <AccountControl className="shrink-0" />
    </header>
  );
}
