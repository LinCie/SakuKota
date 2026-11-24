import { Link } from "@tanstack/react-router";

/**
 * The SakuKota wordmark, linking to the map, with the tagline when asked
 * (only the auth pages show it). `href` keeps a specific map address
 * (kecamatan, filters, place).
 */
export function Wordmark({
  href = "/",
  tagline = false,
}: {
  href?: string;
  tagline?: boolean;
}) {
  return (
    <div className="flex items-center gap-4">
      {/* `href` overrides `to`. */}
      <Link
        to="/"
        href={href}
        className="text-foreground focus-visible:ring-ring/50 shrink-0 rounded-md py-1 text-lg leading-none font-bold tracking-[-0.02em] outline-none focus-visible:ring-3 sm:text-xl"
      >
        Saku<span className="text-primary">Kota</span>
      </Link>
      {tagline ? (
        <p className="text-muted-foreground hidden max-w-[26ch] text-sm leading-snug sm:block">
          Peta bertahan hidup untuk perantau di Jakarta
        </p>
      ) : null}
    </div>
  );
}
