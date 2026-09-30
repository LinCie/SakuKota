import {
  createFileRoute,
  Link,
  Outlet,
  redirect,
  useLocation,
} from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";

import { getSession } from "~/features/auth/session";
import { Wordmark } from "~/shared/components/wordmark";

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
  component: AuthLayout,
});

const plates = [
  { to: "/masuk", label: "Masuk", hint: "Sudah punya akun?" },
  { to: "/daftar", label: "Daftar", hint: "Belum punya akun?" },
] as const;

const plateClass =
  "font-sign text-ink grid h-12 place-items-center rounded-[3px] border-2 text-lg leading-none";

function AuthLayout() {
  const { kembali } = Route.useSearch();
  const pathname = useLocation({ select: (location) => location.pathname });
  // `href` overrides `to`, keeping the map's kecamatan, filters, and place.
  const mapHref = kembali ?? "/";

  return (
    <div className="bg-board text-paint flex min-h-dvh flex-col">
      <header className="px-3 py-3 md:px-5">
        <Wordmark href={mapHref} />
      </header>

      {/* Anchored, not centred: the plates stay put across pages and errors. */}
      <main className="flex flex-1 flex-col items-center px-3 pt-3 pb-10 md:pt-[12dvh] md:pb-16">
        <div className="flex w-full max-w-104 flex-col gap-3">
          <div className="border-ink bg-paint text-ink rounded-[5px] border-2 p-1 motion-safe:transition-[translate] motion-safe:duration-500 motion-safe:ease-[cubic-bezier(0.16,1,0.3,1)] motion-safe:starting:translate-y-3">
            <div className="border-ink/40 flex flex-col gap-5 rounded-[3px] border p-4 sm:p-5">
              <div className="flex flex-col gap-3">
                <div className="grid grid-cols-2 gap-2">
                  {plates.map((plate) =>
                    plate.to === pathname ? (
                      <h1
                        key={plate.to}
                        className={`${plateClass} border-ink bg-sign`}
                      >
                        {plate.label}
                      </h1>
                    ) : (
                      <Link
                        key={plate.to}
                        to={plate.to}
                        search={{ kembali }}
                        className={`${plateClass} border-ink/60 bg-paint hover:border-ink focus-visible:outline-cat border-dashed transition-colors hover:border-solid focus-visible:outline-2 focus-visible:outline-offset-2`}
                      >
                        <span className="sr-only">{plate.hint} </span>
                        {plate.label}
                      </Link>
                    ),
                  )}
                </div>
              </div>

              <Outlet />
            </div>
          </div>

          <Link
            to="/"
            href={mapHref}
            className="text-paint hover:bg-board-deep focus-visible:bg-board-deep focus-visible:outline-sign inline-flex h-10 items-center gap-2 self-start rounded-[3px] px-2 text-sm font-semibold transition-colors focus-visible:outline-2"
          >
            <ArrowLeft
              aria-hidden="true"
              className="size-4"
              strokeWidth={2.5}
            />
            Kembali ke peta
          </Link>
        </div>
      </main>
    </div>
  );
}
