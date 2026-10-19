import { cn } from "~/shared/cn";

import { signButtonClass } from "./menu-board";

/** The full-page board for a page that failed to load, with a link to try again. */
export function ErrorBoard({ title, href }: { title: string; href: string }) {
  return (
    <main className="bg-board grid min-h-dvh place-items-center p-4">
      <div className="border-sign bg-board-deep text-paint flex max-w-md flex-col items-center gap-4 rounded-[5px] border-2 p-6 text-center">
        <span className="font-sign text-sign text-3xl">Rantau</span>
        <p className="font-semibold">{title}</p>
        <p className="text-paint/85 text-sm">
          Server Rantau sedang tidak merespons. Periksa koneksimu, lalu coba
          lagi.
        </p>
        <a
          href={href}
          className={cn(signButtonClass, "focus-visible:outline-sign py-2")}
        >
          Coba lagi
        </a>
      </div>
    </main>
  );
}
