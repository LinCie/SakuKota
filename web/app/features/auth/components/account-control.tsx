import { Link, useLocation } from "@tanstack/react-router";
import { LogOut, MapPinPlus, ShieldCheck, UserRound } from "lucide-react";

import { Button, buttonVariants } from "~/shared/components/ui/button";
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
const iconClass = buttonVariants({ variant: "outline", size: "icon" });

/**
 * The header's account corner: "Masuk" when signed out (returning to the
 * current page afterwards); when signed in, "Usulkan" (not for admins),
 * "Admin" (admins only), the user's first name, and "Keluar". On phones it is
 * a single icon button; signed in, it opens a menu of the same actions.
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
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label={`Akun ${firstName}`}
          className={cn(iconClass, "sm:hidden")}
        >
          <UserRound aria-hidden="true" className="size-5" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-auto min-w-52">
          <DropdownMenuGroup>
            <DropdownMenuLabel className="max-w-60 truncate">
              {session.user.name}
            </DropdownMenuLabel>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          {suggestLink ? (
            <DropdownMenuItem render={<Link to="/usulan" />}>
              <MapPinPlus aria-hidden="true" />
              Usulkan
            </DropdownMenuItem>
          ) : null}
          {adminLink ? (
            <DropdownMenuItem render={<Link to="/admin" />}>
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
      <span className="text-foreground ml-1 max-w-[10ch] truncate text-sm font-semibold max-sm:hidden">
        {firstName}
      </span>
      <Button variant="ghost" onClick={signOut} className="max-sm:hidden">
        <LogOut aria-hidden="true" data-icon="inline-start" />
        Keluar
      </Button>
    </div>
  );
}
