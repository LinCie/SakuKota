import { Menu } from "@base-ui/react/menu";
import { Link, useLocation } from "@tanstack/react-router";
import { LogOut, MapPinPlus, ShieldCheck, UserRound } from "lucide-react";

import { cn } from "~/shared/cn";
import { logout } from "../api";
import { clearSession, isAdmin, useSession } from "../session";

const plateClass =
  "border-paint/35 text-paint hover:border-paint/60 hover:bg-board-deep focus-visible:outline-sign inline-flex h-11 shrink-0 items-center gap-2 rounded-[3px] border-2 px-3 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2";

const itemClass =
  "data-highlighted:bg-paint-deep flex h-11 cursor-pointer items-center gap-2.5 rounded-[3px] px-2.5 text-sm font-bold outline-none";

/**
 * The header's account corner: a "Masuk" plate when signed out (returning to
 * the current page afterwards), a "Usulkan" plate, the user's first name, and
 * "Keluar" when signed in. On phones it is a single icon plate; signed in, it
 * opens a menu of the same actions.
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
        className={cn(
          plateClass,
          "max-sm:size-11 max-sm:justify-center max-sm:px-0",
          className,
        )}
      >
        <UserRound
          aria-hidden="true"
          className="size-4 max-sm:size-5"
          strokeWidth={2.5}
        />
        <span className="sr-only sm:not-sr-only">Masuk</span>
      </Link>
    );
  }

  function signOut() {
    clearSession();
    // Revoking and clearing the refresh cookie is best effort; if it fails,
    // the next page load restores the session from the cookie.
    logout().catch(() => undefined);
  }

  const firstName = session.user.name.split(" ")[0];
  const adminLink = isAdmin(session) && !pathname.startsWith("/admin");
  const suggestLink = pathname !== "/usulan" && !pathname.startsWith("/admin");

  return (
    <div className={cn("flex items-center gap-3", className)}>
      <Menu.Root>
        <Menu.Trigger
          aria-label={`Akun ${firstName}`}
          className={cn(plateClass, "size-11 justify-center px-0 sm:hidden")}
        >
          <UserRound aria-hidden="true" className="size-5" strokeWidth={2.5} />
        </Menu.Trigger>
        <Menu.Portal>
          <Menu.Positioner sideOffset={4} align="end" className="z-50">
            <Menu.Popup className="border-ink/15 bg-paint text-ink min-w-52 rounded-[5px] border p-1 shadow-[0_8px_24px_rgb(26_26_23/0.2)] outline-none">
              <p className="text-ink-soft max-w-60 truncate px-2.5 py-2 text-sm font-semibold">
                {session.user.name}
              </p>
              <div aria-hidden="true" className="bg-ink/15 mx-1 my-1 h-px" />
              {suggestLink ? (
                <Menu.LinkItem
                  render={<Link to="/usulan" />}
                  className={itemClass}
                >
                  <MapPinPlus
                    aria-hidden="true"
                    className="size-4"
                    strokeWidth={2.5}
                  />
                  Usulkan
                </Menu.LinkItem>
              ) : null}
              {adminLink ? (
                <Menu.LinkItem
                  render={<Link to="/admin" />}
                  className={itemClass}
                >
                  <ShieldCheck
                    aria-hidden="true"
                    className="size-4"
                    strokeWidth={2.5}
                  />
                  Admin
                </Menu.LinkItem>
              ) : null}
              <Menu.Item onClick={signOut} className={itemClass}>
                <LogOut
                  aria-hidden="true"
                  className="size-4"
                  strokeWidth={2.5}
                />
                Keluar
              </Menu.Item>
            </Menu.Popup>
          </Menu.Positioner>
        </Menu.Portal>
      </Menu.Root>

      {suggestLink ? (
        <Link to="/usulan" className={cn(plateClass, "max-sm:hidden")}>
          <MapPinPlus aria-hidden="true" className="size-4" strokeWidth={2.5} />
          Usulkan
        </Link>
      ) : null}
      {adminLink ? (
        <Link to="/admin" className={cn(plateClass, "max-sm:hidden")}>
          <ShieldCheck
            aria-hidden="true"
            className="size-4"
            strokeWidth={2.5}
          />
          Admin
        </Link>
      ) : null}
      <span className="text-paint max-w-[10ch] truncate text-sm font-bold max-sm:hidden">
        {firstName}
      </span>
      <button
        type="button"
        onClick={signOut}
        className={cn(plateClass, "max-sm:hidden")}
      >
        <LogOut aria-hidden="true" className="size-4" strokeWidth={2.5} />
        Keluar
      </button>
    </div>
  );
}
