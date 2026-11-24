import { buttonVariants } from "./ui/button";

/** The full-page card for a page that failed to load, with a link to try again. */
export function ErrorBoard({ title, href }: { title: string; href: string }) {
  return (
    <main className="bg-background grid min-h-dvh place-items-center p-4">
      <div className="bg-card ring-foreground/10 flex w-full max-w-md flex-col items-center gap-3 rounded-xl p-6 text-center ring-1">
        <span className="text-xl font-bold tracking-[-0.02em]">
          Saku<span className="text-primary">Kota</span>
        </span>
        <p className="mt-2 font-semibold text-balance">{title}</p>
        <p className="text-muted-foreground text-sm text-pretty">
          Server SakuKota sedang tidak merespons. Periksa koneksimu, lalu coba
          lagi.
        </p>
        <a href={href} className={buttonVariants({ className: "mt-2" })}>
          Coba lagi
        </a>
      </div>
    </main>
  );
}
