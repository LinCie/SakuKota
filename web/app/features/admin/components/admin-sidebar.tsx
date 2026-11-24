import { Link } from "@tanstack/react-router";
import { ArrowLeft, Flag, Inbox, MapIcon, MapPin } from "lucide-react";

import { AccountControl } from "~/features/auth/components/account-control";
import { Badge } from "~/shared/components/ui/badge";
import { Wordmark } from "~/shared/components/wordmark";

const sections = [
  { to: "/admin/places", label: "Tempat", icon: MapPin },
  { to: "/admin/subdistricts", label: "Kecamatan", icon: MapIcon },
  { to: "/admin/suggestions", label: "Usulan", icon: Inbox },
  { to: "/admin/reports", label: "Laporan", icon: Flag },
] as const;

const itemClass =
  "focus-visible:ring-ring/50 inline-flex h-9 shrink-0 items-center gap-2.5 rounded-[10px] px-3 text-sm font-medium transition-colors outline-none focus-visible:ring-3";

/**
 * The admin's frame around the page. From 768px it is a column: the wordmark,
 * the sections, and at the foot the way back to the map and the account. On
 * phones it is a top bar with the account, over a scrolling strip of the
 * sections; the wordmark is the way back.
 */
export function AdminSidebar() {
  return (
    <aside className="bg-background grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-2 border-b px-3 py-2 md:sticky md:top-0 md:flex md:h-dvh md:flex-col md:items-stretch md:gap-6 md:border-r md:border-b-0 md:px-4 md:py-6">
      <div className="flex items-center gap-2.5 md:px-1">
        <Wordmark />
        <Badge variant="outline">Admin</Badge>
      </div>
      <nav
        aria-label="Menu admin"
        className="col-span-full -mx-3 flex gap-1 overflow-x-auto px-3 md:mx-0 md:flex-col md:overflow-visible md:px-0"
      >
        {sections.map(({ to, label, icon: Icon }) => (
          <Link
            key={to}
            to={to}
            className={`${itemClass} text-foreground not-data-[status=active]:hover:bg-muted`}
            activeProps={{ className: "bg-pandan-soft text-pandan-deep" }}
          >
            <Icon aria-hidden="true" className="size-4" />
            {label}
          </Link>
        ))}
      </nav>
      <div className="col-start-2 row-start-1 md:mt-auto md:flex md:flex-col md:items-start md:gap-3 md:border-t md:pt-4">
        <Link
          to="/"
          className={`${itemClass} text-muted-foreground hover:bg-muted hover:text-foreground w-full max-md:hidden`}
        >
          <ArrowLeft aria-hidden="true" className="size-4" />
          Kembali ke peta
        </Link>
        <AccountControl />
      </div>
    </aside>
  );
}
