import { ArrowLeft, X } from "lucide-react";
import { useEffect, useRef, type ReactNode } from "react";

import { MissingSlot } from "~/shared/components/missing-slot";
import type { Place } from "../types";
import { CategoryBadge, CodePlate } from "./category-sign";

// -----------------------------------------------------------------------------
// List
// -----------------------------------------------------------------------------

type PlaceListProps = {
  places: Place[];
  onSelect: (placeId: number) => void;
  /** Shown when `places` is empty. */
  emptyMessage: string;
};

export function PlaceList({ places, onSelect, emptyMessage }: PlaceListProps) {
  if (places.length === 0) {
    return (
      <p className="text-ink-soft px-4 py-6 text-sm md:px-5">{emptyMessage}</p>
    );
  }

  return (
    <ul>
      {places.map((place) => {
        const category = place.categories[0];

        return (
          <li
            key={place.id}
            className="border-ink/20 border-b border-dashed last:border-b-0"
          >
            <button
              type="button"
              onClick={() => onSelect(place.id)}
              className="hover:bg-sign-soft/60 focus-visible:bg-sign-soft/60 focus-visible:outline-cat flex w-full items-start gap-3 px-4 py-3 text-left transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 md:px-5"
            >
              {category ? (
                <CodePlate category={category} className="mt-0.5" />
              ) : null}
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
        );
      })}
    </ul>
  );
}

// -----------------------------------------------------------------------------
// Details (the menu board)
// -----------------------------------------------------------------------------

/** A menu-board row: label, dotted leader, value. */
function LeaderRow({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-baseline gap-2 py-2">
      <dt className="text-ink-soft shrink-0 text-sm font-semibold">{label}</dt>
      <span
        aria-hidden="true"
        className="border-ink/25 min-w-6 flex-1 -translate-y-1 border-b-2 border-dotted"
      />
      <dd className="text-ink max-w-[65%] text-right font-semibold tabular-nums">
        {children}
      </dd>
    </div>
  );
}

type PlaceDetailsProps = {
  place: Place;
  onClose: () => void;
};

export function PlaceDetails({ place, onClose }: PlaceDetailsProps) {
  const headingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    headingRef.current?.focus({ preventScroll: true });
  }, [place.id]);

  return (
    <article aria-labelledby="place-name" className="flex flex-col">
      <div className="border-ink flex items-center justify-between gap-2 border-b-2 px-2 py-2 md:px-3">
        <button
          type="button"
          onClick={onClose}
          className="text-ink hover:bg-sign-soft focus-visible:bg-sign-soft focus-visible:outline-cat hidden items-center gap-2 rounded-[3px] px-2 py-1.5 text-sm font-semibold transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 md:inline-flex"
        >
          <ArrowLeft aria-hidden="true" className="size-4" strokeWidth={2.5} />
          Daftar tempat
        </button>
        <span
          aria-hidden="true"
          className="bg-ink/20 mx-auto h-1.5 w-12 rounded-full md:hidden"
        />
        <button
          type="button"
          onClick={onClose}
          aria-label="Tutup detail tempat"
          className="text-ink hover:bg-sign-soft focus-visible:bg-sign-soft focus-visible:outline-cat grid size-9 place-items-center rounded-[3px] transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 md:hidden"
        >
          <X aria-hidden="true" className="size-5" strokeWidth={2.5} />
        </button>
      </div>

      <div className="p-3 md:p-4">
        <div className="border-ink bg-paint rounded-[5px] border-2 p-1">
          <div className="border-ink/40 flex flex-col gap-4 rounded-[3px] border px-4 py-4">
            <div className="flex flex-wrap gap-x-4 gap-y-2">
              {place.categories.map((category) => (
                <CategoryBadge key={category.id} category={category} />
              ))}
            </div>

            <h2
              id="place-name"
              ref={headingRef}
              tabIndex={-1}
              className="font-sign text-ink text-2xl leading-tight text-balance outline-none"
            >
              {place.name}
            </h2>

            {place.short_description ? (
              <p className="text-ink max-w-prose text-pretty">
                {place.short_description}
              </p>
            ) : (
              <MissingSlot>Deskripsi tidak tersedia</MissingSlot>
            )}

            <dl className="border-ink border-t-2 pt-1">
              <LeaderRow label="Jam buka">
                {place.operating_hours ?? (
                  <MissingSlot>Jam buka tidak tersedia</MissingSlot>
                )}
              </LeaderRow>
              <LeaderRow label="Kisaran harga">
                {place.estimated_price_range ?? (
                  <MissingSlot>Harga tidak tersedia</MissingSlot>
                )}
              </LeaderRow>
              <div className="border-ink/25 flex flex-col gap-1 border-t border-dashed py-2">
                <dt className="text-ink-soft text-sm font-semibold">Alamat</dt>
                <dd className="text-ink">
                  {place.address ?? (
                    <MissingSlot>Alamat tidak tersedia</MissingSlot>
                  )}
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </div>
    </article>
  );
}
