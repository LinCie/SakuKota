import { Link, Outlet, useLocation } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";

import { cn } from "~/shared/cn";
import { Wordmark } from "~/shared/components/wordmark";

const plates = [
  { to: "/masuk", label: "Masuk", hint: "Sudah punya akun?" },
  { to: "/daftar", label: "Daftar", hint: "Belum punya akun?" },
] as const;

const segmentClass =
  "grid h-9 place-items-center rounded-full text-sm leading-none font-semibold";

type AuthShellProps = {
  /** Where the map links return to: the page the visitor came from. */
  kembali?: string;
};

/** The auth frame: the wordmark, the masuk/daftar switch, and the page. */
export function AuthShell({ kembali }: AuthShellProps) {
  const pathname = useLocation({ select: (location) => location.pathname });
  // `href` overrides `to`, keeping the map's kecamatan, filters, and place.
  const mapHref = kembali ?? "/";

  return (
    <div className="bg-background text-foreground flex min-h-dvh flex-col">
      <header className="px-3 py-2.5 md:px-5 md:py-3.5">
        <Wordmark href={mapHref} tagline />
      </header>

      {/* Anchored, not centred: the switch stays put across pages and errors. */}
      <main className="flex flex-1 flex-col items-center px-3 pt-3 pb-10 md:pt-[12dvh] md:pb-16">
        <div className="flex w-full max-w-104 flex-col gap-3">
          <div className="bg-card ring-foreground/10 flex flex-col gap-5 rounded-xl p-5 ring-1 motion-safe:transition-[translate,opacity] motion-safe:duration-400 motion-safe:ease-[cubic-bezier(0.16,1,0.3,1)] sm:p-6 motion-safe:starting:translate-y-2 motion-safe:starting:opacity-0">
            <div className="bg-muted grid grid-cols-2 rounded-full p-1">
              {plates.map((plate) =>
                plate.to === pathname ? (
                  <h1
                    key={plate.to}
                    className={cn(
                      segmentClass,
                      "bg-card text-foreground shadow-sm",
                    )}
                  >
                    {plate.label}
                  </h1>
                ) : (
                  <Link
                    key={plate.to}
                    to={plate.to}
                    search={{ kembali }}
                    className={cn(
                      segmentClass,
                      "text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 transition-colors outline-none focus-visible:ring-3",
                    )}
                  >
                    <span className="sr-only">{plate.hint} </span>
                    {plate.label}
                  </Link>
                ),
              )}
            </div>

            <Outlet />
          </div>

          <Link
            to="/"
            href={mapHref}
            className="text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:ring-ring/50 inline-flex h-10 items-center gap-2 self-start rounded-full px-3 text-sm font-medium transition-colors outline-none focus-visible:ring-3"
          >
            <ArrowLeft aria-hidden="true" className="size-4" />
            Kembali ke peta
          </Link>
        </div>
      </main>
    </div>
  );
}
