import { Loader2 } from "lucide-react";

/** Shown while a page's data loads. */
export function LoadingState() {
  return (
    <div
      role="status"
      className="text-ink-soft flex items-center gap-2 py-10 text-sm font-semibold"
    >
      <Loader2
        aria-hidden="true"
        className="size-4 motion-safe:animate-spin"
        strokeWidth={2.5}
      />
      Memuat…
    </div>
  );
}
