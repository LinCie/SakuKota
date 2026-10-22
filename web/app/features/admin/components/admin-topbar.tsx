import { Link } from "@tanstack/react-router";

import { AccountControl } from "~/features/auth/components/account-control";
import { Badge } from "~/shared/components/badge";

/** The admin's top bar: the console label, a way back to the map, and the account. */
export function AdminTopbar() {
  return (
    <header className="bg-ink text-paint sticky top-0 z-10 flex h-14 items-center justify-between gap-3 px-4 md:px-6">
      <div className="flex min-w-0 items-center gap-4">
        <Badge className="border-sign text-sign">Admin</Badge>
        <Link
          to="/"
          className="text-paint/80 hover:text-paint focus-visible:outline-sign rounded-[3px] text-sm font-semibold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
        >
          Kembali ke peta
        </Link>
      </div>
      <AccountControl className="shrink-0" />
    </header>
  );
}
