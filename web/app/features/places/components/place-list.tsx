import { useEffect, useEffectEvent, useRef } from "react";

import type { Place } from "../types";
import { CategoryTile } from "./category-sign";

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

/** The places as a list; scrolling to the end loads more, and picking one opens its details. */
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
      <p className="text-muted-foreground px-4 py-6 text-sm md:px-5">
        {emptyMessage}
      </p>
    );
  }

  return (
    <ul>
      {places.map((place) => (
        <li key={place.id} className="border-b last:border-b-0">
          <button
            type="button"
            onClick={() => onSelect(place)}
            className="hover:bg-muted focus-visible:bg-muted focus-visible:ring-ring/50 flex w-full items-start gap-3 px-4 py-3 text-left transition-colors outline-none focus-visible:ring-3 focus-visible:ring-inset md:px-5"
          >
            <CategoryTile category={place.category} className="mt-0.5" />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="text-foreground font-semibold">
                {place.name}
              </span>
              <span className="text-muted-foreground truncate text-sm">
                {place.address ?? "Alamat tidak tersedia"}
              </span>
            </span>
            {place.estimated_price_range ? (
              <span className="text-foreground mt-0.5 shrink-0 text-right text-sm font-medium tabular-nums">
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
        <li
          role="status"
          className="text-muted-foreground px-4 py-3 text-sm md:px-5"
        >
          Memuat tempat…
        </li>
      ) : null}
      {loadMoreFailed && !loadingMore ? (
        <li className="px-4 py-3 text-sm md:px-5">
          <button
            type="button"
            onClick={onLoadMore}
            className="text-primary focus-visible:ring-ring/50 rounded-md font-medium underline underline-offset-4 outline-none focus-visible:ring-3"
          >
            Gagal memuat. Coba lagi
          </button>
        </li>
      ) : null}
    </ul>
  );
}
