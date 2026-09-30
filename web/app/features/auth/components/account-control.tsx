import { Link, useLocation } from "@tanstack/react-router";
import { LogOut, UserRound } from "lucide-react";

import { logout } from "../api";
import { clearSession, useSession } from "../session";

const plateClass =
  "border-paint/35 text-paint hover:border-paint/60 hover:bg-board-deep focus-visible:outline-sign inline-flex h-11 shrink-0 items-center gap-2 rounded-[3px] border-2 px-3 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2";

/**
 * The header's account corner: a "Masuk" plate when signed out (returning to
 * the current page afterwards), the user's first name and "Keluar" when
 * signed in.
 */
export function AccountControl({ className = "" }: { className?: string }) {
  const session = useSession();
  const href = useLocation({ select: (location) => location.href });

  if (!session) {
    return (
      <Link
        to="/masuk"
        search={{ kembali: href === "/" ? undefined : href }}
        className={`${plateClass} ${className}`}
      >
        <UserRound aria-hidden="true" className="size-4" strokeWidth={2.5} />
        Masuk
      </Link>
    );
  }

  function signOut() {
    if (!session) return;
    clearSession();
    // Revoking the refresh token is best effort; the user is signed out here
    // either way.
    logout(session.refresh_token).catch(() => undefined);
  }

  return (
    <div className={`flex items-center gap-3 ${className}`}>
      <span className="text-paint max-w-[10ch] truncate text-sm font-bold">
        {session.user.name.split(" ")[0]}
      </span>
      <button type="button" onClick={signOut} className={plateClass}>
        <LogOut aria-hidden="true" className="size-4" strokeWidth={2.5} />
        Keluar
      </button>
    </div>
  );
}
