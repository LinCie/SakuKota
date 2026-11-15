import { Link } from "@tanstack/react-router";

/**
 * The RANTAU plate and tagline on the green board, linking to the map.
 * `href` keeps a specific map address (kecamatan, filters, place).
 */
export function Wordmark({
  href = "/",
  tagline = true,
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
        className="bg-board-deep focus-visible:outline-sign shrink-0 rounded-[5px] p-1 focus-visible:outline-2 focus-visible:outline-offset-2"
      >
        <span className="border-sign font-sign text-sign block rounded-[3px] border-2 px-2.5 pt-1.5 pb-1 text-xl leading-none tracking-wide sm:px-3 sm:text-2xl">
          Rantau
        </span>
      </Link>
      {tagline ? (
        <p className="text-paint/85 hidden max-w-[26ch] text-sm leading-snug sm:block">
          Peta bertahan hidup untuk perantau di Jakarta
        </p>
      ) : null}
    </div>
  );
}
