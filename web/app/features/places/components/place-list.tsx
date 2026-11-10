import { useEffect, useEffectEvent, useRef } from "react";

import type { Place } from "../types";
import { CodePlate } from "./category-sign";

type PlaceListProps = {
  places: Place[];
  onSelect: (place: Place) => void;
  /** Shown when `places` is empty. */
  emptyMessage: string;
  /** Whether more places can be loaded below the last one. */
  hasMore: boolean;
  loadingMore: boolean;
  /** The last attempt to load more failed; the user retries by hand. */
  loadMoreFailed: boolean;
  onLoadMore: () => void;
};

/** The places on the board as a list; scrolling to the end loads more, and picking one opens its details. */
export function PlaceList({
  places,
  onSelect,
  emptyMessage,
  hasMore,
  loadingMore,
  loadMoreFailed,
  onLoadMore,
}: PlaceListProps) {
  const sentinelRef = useRef<HTMLLIElement>(null);
  const loadMore = useEffectEvent(onLoadMore);
  const waiting = hasMore && !loadingMore && !loadMoreFailed;

  // Observing again after each page reports a sentinel that is still in view.
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !waiting) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) loadMore();
      },
      { rootMargin: "240px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [waiting, places.length]);

  if (places.length === 0) {
    return (
      <p className="text-ink-soft px-4 py-6 text-sm md:px-5">{emptyMessage}</p>
    );
  }

  return (
    <ul>
      {places.map((place) => (
        <li
          key={place.id}
          className="border-ink/20 border-b border-dashed last:border-b-0"
        >
          <button
            type="button"
            onClick={() => onSelect(place)}
            className="hover:bg-sign-soft/60 focus-visible:bg-sign-soft/60 focus-visible:outline-cat flex w-full items-start gap-3 px-4 py-3 text-left transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 md:px-5"
          >
            <CodePlate category={place.category} className="mt-0.5" />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="text-ink font-semibold">{place.name}</span>
              <span className="text-ink-soft truncate text-sm">
                {place.address ?? "Alamat tidak tersedia"}
              </span>
            </span>
            {place.estimated_price_range ? (
              <span className="text-ink mt-0.5 shrink-0 text-right text-sm font-semibold tabular-nums">
                {place.estimated_price_range}
              </span>
            ) : null}
          </button>
        </li>
      ))}
      {waiting ? (
        <li ref={sentinelRef} aria-hidden="true" className="h-px" />
      ) : null}
      {loadingMore ? (
        <li role="status" className="text-ink-soft px-4 py-3 text-sm md:px-5">
          Memuat tempat…
        </li>
      ) : null}
      {loadMoreFailed && !loadingMore ? (
        <li className="px-4 py-3 text-sm md:px-5">
          <button
            type="button"
            onClick={onLoadMore}
            className="text-cat font-semibold underline"
          >
            Gagal memuat. Coba lagi
          </button>
        </li>
      ) : null}
    </ul>
  );
}
