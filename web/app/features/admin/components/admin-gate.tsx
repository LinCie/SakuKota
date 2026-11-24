import { Link } from "@tanstack/react-router";

import { buttonVariants } from "~/shared/components/ui/button";
import { Wordmark } from "~/shared/components/wordmark";

/** Shown in place of the console to anyone who is not an admin. */
export function AdminGate({ signedIn }: { signedIn: boolean }) {
  return (
    <div className="bg-background text-foreground flex min-h-dvh flex-col">
      <header className="bg-background border-b px-4 py-3">
        <Wordmark tagline={false} />
      </header>
      <main className="grid flex-1 place-items-center p-4">
        <div className="bg-card ring-foreground/10 flex w-full max-w-sm flex-col gap-4 rounded-xl p-6 ring-1">
          <h1 className="text-xl font-semibold">
            {signedIn ? "Khusus admin" : "Masuk dulu"}
          </h1>
          <p className="text-muted-foreground text-sm text-pretty">
            {signedIn
              ? "Akun ini belum punya akses admin. Masuk dengan akun admin untuk mengelola data."
              : "Masuk dengan akun admin untuk mengelola tempat, kecamatan, dan kategori."}
          </p>
          {signedIn ? null : (
            <Link
              to="/masuk"
              search={{ kembali: "/admin" }}
              className={buttonVariants()}
            >
              Masuk
            </Link>
          )}
          <Link
            to="/"
            className={buttonVariants({
              variant: "ghost",
              size: "sm",
              className: "text-muted-foreground self-center",
            })}
          >
            Kembali ke peta
          </Link>
        </div>
      </main>
    </div>
  );
}
