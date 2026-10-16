import { ArrowLeft, Check, X } from "lucide-react";
import { useEffect, useRef, type ReactNode } from "react";

import { cn } from "~/shared/cn";
import { MissingSlot, orMissing } from "~/shared/components/missing-slot";
import {
  LeaderRow,
  menuButtonClass,
  StackedRow,
} from "~/shared/components/menu-board";
import { formatDate, formatNumber } from "~/shared/format";
import type { CategoryField, Place } from "../types";
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
      {places.map((place) => (
        <li
          key={place.id}
          className="border-ink/20 border-b border-dashed last:border-b-0"
        >
          <button
            type="button"
            onClick={() => onSelect(place.id)}
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
    </ul>
  );
}

// -----------------------------------------------------------------------------
// Details (the menu board)
// -----------------------------------------------------------------------------

function isRange(value: unknown): value is { min: number; max: number } {
  return (
    typeof value === "object" &&
    value !== null &&
    "min" in value &&
    "max" in value &&
    typeof value.min === "number" &&
    typeof value.max === "number"
  );
}

/**
 * Renders a custom field value by its data type, or null when the value is
 * missing or doesn't match the type.
 */
function formatAttribute(field: CategoryField, value: unknown): ReactNode {
  switch (field.data_type) {
    case "text":
    case "select":
      return typeof value === "string" && value.trim() !== "" ? value : null;
    case "number":
      return typeof value === "number" ? formatNumber(value) : null;
    case "boolean": {
      if (typeof value !== "boolean") return null;
      const Icon = value ? Check : X;
      return (
        // The word sets the baseline so the row's leader lines up with it.
        <span className="inline-flex items-baseline gap-1">
          <Icon
            aria-hidden="true"
            className="size-4 self-center"
            strokeWidth={2.5}
          />
          {value ? "Ya" : "Tidak"}
        </span>
      );
    }
    case "range":
      if (!isRange(value)) return null;
      return value.min === value.max
        ? formatNumber(value.min)
        : `${formatNumber(value.min)}–${formatNumber(value.max)}`;
    case "date": {
      const date = typeof value === "string" ? new Date(value) : null;
      return date && !Number.isNaN(date.getTime()) ? formatDate(date) : null;
    }
    default:
      // A type this frontend doesn't know yet: show plain values as-is.
      return typeof value === "string" || typeof value === "number"
        ? String(value)
        : null;
  }
}

function attributeValue(field: CategoryField, place: Place): ReactNode {
  return orMissing(formatAttribute(field, place.attributes[field.key]));
}

type PlaceDetailsProps = {
  place: Place;
  /** Custom fields of the place's category, sorted by `sort_order`. */
  fields: CategoryField[];
  onClose: () => void;
};

export function PlaceDetails({ place, fields, onClose }: PlaceDetailsProps) {
  const headingRef = useRef<HTMLHeadingElement>(null);

  // Short values sit on leader rows; free text follows as full-width blocks,
  // so `sort_order` applies within each group.
  const leaderFields = fields.filter((field) => field.data_type !== "text");
  const textFields = fields.filter((field) => field.data_type === "text");

  useEffect(() => {
    headingRef.current?.focus({ preventScroll: true });
  }, [place.id]);

  return (
    <article aria-labelledby="place-name" className="flex flex-col">
      <div className="border-ink flex items-center justify-between gap-2 border-b-2 px-2 py-2 md:px-3">
        <button
          type="button"
          onClick={onClose}
          className={cn(
            menuButtonClass,
            "hidden items-center gap-2 rounded-[3px] px-2 py-1.5 text-sm font-semibold md:inline-flex",
          )}
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
          className={cn(
            menuButtonClass,
            "grid size-9 place-items-center rounded-[3px] md:hidden",
          )}
        >
          <X aria-hidden="true" className="size-5" strokeWidth={2.5} />
        </button>
      </div>

      <div className="p-3 md:p-4">
        <div className="border-ink bg-paint rounded-[5px] border-2 p-1">
          <div className="border-ink/40 flex flex-col gap-4 rounded-[3px] border px-4 py-4">
            <div>
              <CategoryBadge category={place.category} />
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
                {orMissing(place.operating_hours, "Jam buka tidak tersedia")}
              </LeaderRow>
              <LeaderRow label="Kisaran harga">
                {orMissing(place.estimated_price_range, "Harga tidak tersedia")}
              </LeaderRow>
              {leaderFields.map((field) => (
                <LeaderRow key={field.id} label={field.label}>
                  {attributeValue(field, place)}
                </LeaderRow>
              ))}
              {textFields.map((field) => (
                <StackedRow key={field.id} label={field.label}>
                  {attributeValue(field, place)}
                </StackedRow>
              ))}
              <StackedRow label="Alamat">
                {orMissing(place.address, "Alamat tidak tersedia")}
              </StackedRow>
            </dl>
          </div>
        </div>
      </div>
    </article>
  );
}
