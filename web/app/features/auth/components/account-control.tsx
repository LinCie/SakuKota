import { Link, useLocation } from "@tanstack/react-router";
import { LogOut, MapPinPlus, ShieldCheck, UserRound } from "lucide-react";

import { cn } from "~/shared/cn";
import { logout } from "../api";
import { useSession } from "../hooks";
import { clearSession, isAdmin } from "../session";

const plateClass =
  "border-paint/35 text-paint hover:border-paint/60 hover:bg-board-deep focus-visible:outline-sign inline-flex h-11 shrink-0 items-center gap-2 rounded-[3px] border-2 px-3 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2";

/**
 * The header's account corner: a "Masuk" plate when signed out (returning to
 * the current page afterwards), a "Usulkan" plate, the user's first name, and
 * "Keluar" when signed in.
 */
export function AccountControl({ className }: { className?: string }) {
  const session = useSession();
  const href = useLocation({ select: (location) => location.href });
  const pathname = useLocation({ select: (location) => location.pathname });

  if (!session) {
    return (
      <Link
        to="/masuk"
        search={{ kembali: href === "/" ? undefined : href }}
        className={cn(plateClass, className)}
      >
        <UserRound aria-hidden="true" className="size-4" strokeWidth={2.5} />
        Masuk
      </Link>
    );
  }

  function signOut() {
    clearSession();
    // Revoking and clearing the refresh cookie is best effort; if it fails,
    // the next page load restores the session from the cookie.
    logout().catch(() => undefined);
  }

  // The admin plate is icon-only on phones, where it replaces the name.
  const admin = isAdmin(session);
  const adminLink = admin && !pathname.startsWith("/admin");
  // Like the admin plate, the suggest plate is icon-only on phones.
  const suggestLink = pathname !== "/usulan" && !pathname.startsWith("/admin");

  return (
    <div className={cn("flex items-center gap-3", className)}>
      {suggestLink ? (
        <Link to="/usulan" className={plateClass}>
          <MapPinPlus aria-hidden="true" className="size-4" strokeWidth={2.5} />
          <span className="sr-only sm:not-sr-only">Usulkan</span>
        </Link>
      ) : null}
      {adminLink ? (
        <Link to="/admin" className={plateClass}>
          <ShieldCheck
            aria-hidden="true"
            className="size-4"
            strokeWidth={2.5}
          />
          <span className="sr-only sm:not-sr-only">Admin</span>
        </Link>
      ) : null}
      <span
        className={cn(
          "text-paint max-w-[10ch] truncate text-sm font-bold",
          admin && "hidden sm:block",
        )}
      >
        {session.user.name.split(" ")[0]}
      </span>
      <button type="button" onClick={signOut} className={plateClass}>
        <LogOut aria-hidden="true" className="size-4" strokeWidth={2.5} />
        Keluar
      </button>
    </div>
  );
}
