import { Link, useLocation } from "@tanstack/react-router";
import { LogOut, MapPinPlus, ShieldCheck, UserRound } from "lucide-react";

import { buttonVariants } from "~/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "~/shared/components/ui/dropdown-menu";
import { cn } from "~/shared/cn";
import { logout } from "../api";
import { clearSession, isAdmin, useSession } from "../session";

const outlineClass = buttonVariants({ variant: "outline" });

/**
 * The header's account corner: "Masuk" when signed out (returning to the
 * current page afterwards); when signed in, "Usulkan" (not for admins),
 * "Admin" (admins only), and a menu under the user's first name with
 * "Profil" and "Keluar". On phones it is a single icon button whose menu also
 * holds "Usulkan" and "Admin".
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
          outlineClass,
          "max-sm:size-11 max-sm:px-0",
          "shrink-0",
          className,
        )}
      >
        <UserRound
          aria-hidden="true"
          data-icon="inline-start"
          className="max-sm:size-5"
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
  const admin = isAdmin(session);
  const adminLink = admin && !pathname.startsWith("/admin");
  const suggestLink =
    !admin && pathname !== "/usulan" && !pathname.startsWith("/admin");

  return (
    <div className={cn("flex shrink-0 items-center gap-2", className)}>
      {suggestLink ? (
        <Link to="/usulan" className={cn(outlineClass, "max-sm:hidden")}>
          <MapPinPlus aria-hidden="true" data-icon="inline-start" />
          Usulkan
        </Link>
      ) : null}
      {adminLink ? (
        <Link to="/admin" className={cn(outlineClass, "max-sm:hidden")}>
          <ShieldCheck aria-hidden="true" data-icon="inline-start" />
          Admin
        </Link>
      ) : null}
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label={`Akun ${firstName}`}
          className={cn(outlineClass, "max-sm:size-11 max-sm:px-0")}
        >
          <UserRound
            aria-hidden="true"
            data-icon="inline-start"
            className="max-sm:size-5"
          />
          <span className="max-w-[10ch] truncate max-sm:sr-only">
            {firstName}
          </span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-auto min-w-52">
          <DropdownMenuGroup>
            <DropdownMenuLabel className="max-w-60 truncate">
              {session.user.name}
            </DropdownMenuLabel>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          {pathname !== "/profil" ? (
            <DropdownMenuItem render={<Link to="/profil" />}>
              <UserRound aria-hidden="true" />
              Profil
            </DropdownMenuItem>
          ) : null}
          {suggestLink ? (
            <DropdownMenuItem
              render={<Link to="/usulan" />}
              className="sm:hidden"
            >
              <MapPinPlus aria-hidden="true" />
              Usulkan
            </DropdownMenuItem>
          ) : null}
          {adminLink ? (
            <DropdownMenuItem
              render={<Link to="/admin" />}
              className="sm:hidden"
            >
              <ShieldCheck aria-hidden="true" />
              Admin
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuItem onClick={signOut}>
            <LogOut aria-hidden="true" />
            Keluar
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
