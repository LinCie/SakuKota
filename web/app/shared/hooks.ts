import { useRef, useState } from "react";

import { slugify } from "~/shared/text";

export type Position = { lat: number; lng: number };

/** Reads the device's current position once, handing it to `onFound`. */
export function useCurrentPosition() {
  const [locating, setLocating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function locate(onFound: (position: Position) => void) {
    setError(null);
    if (!("geolocation" in navigator)) {
      setError("Perangkat ini tidak mendukung lokasi.");
      return;
    }
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        setLocating(false);
        onFound({ lat: coords.latitude, lng: coords.longitude });
      },
      () => {
        setLocating(false);
        setError("Lokasi tidak terbaca. Izinkan akses lokasi, lalu coba lagi.");
      },
      { enableHighAccuracy: true, timeout: 10_000 },
    );
  }

  return { locating, error, locate };
}

/** Open state for a drawer or dialog. */
export function useDisclosure() {
  const [open, setOpen] = useState(false);
  return { open, setOpen };
}

/**
 * Keeps a slug in step with its name until the slug is edited by hand. Keep
 * the returned object for the life of one form; it resets when the form unmounts.
 */
export function useSlugFollow(followSlug: boolean) {
  const edited = useRef(!followSlug);

  return {
    /** The slug for a name, or null once the slug was edited by hand. */
    slugFor: (name: string): string | null =>
      edited.current ? null : slugify(name),
    /** Records a hand edit, which stops the slug following the name. */
    markEdited: () => {
      edited.current = true;
    },
  };
}

/**
 * Filters table rows by a search term, matched case-insensitively against the
 * text `textOf` returns for each row.
 */
export function useTableQuery<T>(rows: T[], textOf: (row: T) => string) {
  const [query, setQuery] = useState("");
  const needle = query.trim().toLowerCase();

  const filtered =
    needle === ""
      ? rows
      : rows.filter((row) => textOf(row).toLowerCase().includes(needle));

  return { query, setQuery, rows: filtered };
}
