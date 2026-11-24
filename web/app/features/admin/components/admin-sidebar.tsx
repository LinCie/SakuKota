import { Link } from "@tanstack/react-router";
import { Flag, Inbox, MapPin } from "lucide-react";

import { Wordmark } from "~/shared/components/wordmark";

const sections = [
  { to: "/admin/places", label: "Tempat", icon: MapPin },
  { to: "/admin/suggestions", label: "Usulan", icon: Inbox },
  { to: "/admin/reports", label: "Laporan", icon: Flag },
] as const;

/** The admin's navigation: a column on desktop, a scrolling strip on phones. */
export function AdminSidebar() {
  return (
    <aside className="bg-background flex flex-col gap-4 border-b px-4 py-3 md:sticky md:top-0 md:h-dvh md:border-r md:border-b-0 md:py-5">
      <Wordmark tagline={false} />
      <nav
        aria-label="Menu admin"
        className="flex gap-1 overflow-x-auto md:flex-col md:overflow-visible"
      >
        {sections.map(({ to, label, icon: Icon }) => (
          <Link
            key={to}
            to={to}
            className="text-foreground not-data-[status=active]:hover:bg-muted focus-visible:ring-ring/50 inline-flex h-9 shrink-0 items-center gap-2.5 rounded-[10px] px-3 text-sm font-medium transition-colors outline-none focus-visible:ring-3"
            activeProps={{ className: "bg-pandan-soft text-pandan-deep" }}
          >
            <Icon aria-hidden="true" className="size-4" />
            {label}
          </Link>
        ))}
      </nav>
    </aside>
  );
}
