import { Link } from "@tanstack/react-router";
import { Inbox, MapPin } from "lucide-react";

import { Wordmark } from "~/shared/components/wordmark";

const sections = [
  { to: "/admin/places", label: "Tempat", icon: MapPin },
  { to: "/admin/suggestions", label: "Usulan", icon: Inbox },
] as const;

/** The admin's navigation: a column on desktop, a scrolling strip on phones. */
export function AdminSidebar() {
  return (
    <aside className="border-ink/15 bg-paint-deep flex flex-col gap-4 border-b px-4 py-3 md:sticky md:top-0 md:h-dvh md:border-r md:border-b-0 md:py-5">
      <Wordmark tagline={false} />
      <nav
        aria-label="Menu admin"
        className="flex gap-1 overflow-x-auto md:flex-col md:overflow-visible"
      >
        {sections.map(({ to, label, icon: Icon }) => (
          <Link
            key={to}
            to={to}
            className="text-ink not-data-[status=active]:hover:bg-paint focus-visible:outline-cat inline-flex h-9 shrink-0 items-center gap-2.5 rounded-[3px] px-3 text-sm font-semibold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
            activeProps={{ className: "bg-board text-paint" }}
          >
            <Icon aria-hidden="true" className="size-4" strokeWidth={2.5} />
            {label}
          </Link>
        ))}
      </nav>
    </aside>
  );
}
