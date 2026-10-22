import { Link } from "@tanstack/react-router";

import { Button } from "~/shared/components/button";
import { Wordmark } from "~/shared/components/wordmark";

/** Shown in place of the console to anyone who is not an admin. */
export function AdminGate({ signedIn }: { signedIn: boolean }) {
  return (
    <div className="bg-paint text-ink flex min-h-dvh flex-col">
      <header className="bg-ink px-4 py-3">
        <Wordmark tagline={false} />
      </header>
      <main className="grid flex-1 place-items-center p-4">
        <div className="border-ink/15 bg-paint flex w-full max-w-sm flex-col gap-4 rounded-[5px] border p-6">
          <h1 className="text-xl font-bold">
            {signedIn ? "Khusus admin" : "Masuk dulu"}
          </h1>
          <p className="text-ink-soft text-sm text-pretty">
            {signedIn
              ? "Akun ini belum punya akses admin. Masuk dengan akun admin untuk mengelola data."
              : "Masuk dengan akun admin untuk mengelola tempat, kecamatan, dan kategori."}
          </p>
          {signedIn ? null : (
            <Button
              variant="primary"
              render={<Link to="/masuk" search={{ kembali: "/admin" }} />}
            >
              Masuk
            </Button>
          )}
          <Link
            to="/"
            className="text-ink-soft hover:text-ink text-sm font-semibold"
          >
            Kembali ke peta
          </Link>
        </div>
      </main>
    </div>
  );
}
