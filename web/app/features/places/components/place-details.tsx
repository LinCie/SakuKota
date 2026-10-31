import { ArrowLeft, Star, X } from "lucide-react";
import { useEffect, useRef } from "react";

import { cn } from "~/shared/cn";
import { ImageFrame } from "~/shared/components/image-frame";
import { MissingSlot, orMissing } from "~/shared/components/missing-slot";
import { formatNumber, formatRating } from "~/shared/format";
import {
  LeaderRow,
  menuButtonClass,
  StackedRow,
} from "~/shared/components/menu-board";
import { attributeValue } from "../attribute-format";
import type { CategoryField, Place } from "../types";
import { CategoryBadge } from "./category-sign";

type PlaceDetailsProps = {
  place: Place;
  /** Custom fields of the place's category, sorted by `sort_order`. */
  fields: CategoryField[];
  onClose: () => void;
};

/** One place on the menu board: its details, with a way back to the list. */
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

            <ImageFrame
              src={place.image_url}
              alt={`Foto ${place.name}`}
              missing="Foto tidak tersedia"
              aspect="aspect-4/3"
            />

            {place.short_description ? (
              <p className="text-ink max-w-prose text-pretty">
                {place.short_description}
              </p>
            ) : (
              <MissingSlot>Deskripsi tidak tersedia</MissingSlot>
            )}

            <dl className="border-ink border-t-2 pt-1">
              <LeaderRow label="Rating">
                {place.rating_average === null ? (
                  <MissingSlot>Belum ada ulasan</MissingSlot>
                ) : (
                  <span className="inline-flex items-center gap-1.5">
                    <Star
                      aria-hidden="true"
                      className="fill-sign text-ink size-4 shrink-0"
                      strokeWidth={2.5}
                    />
                    {formatRating(place.rating_average)}
                    <span className="text-ink-soft text-sm font-normal">
                      ({formatNumber(place.rating_count)} ulasan)
                    </span>
                  </span>
                )}
              </LeaderRow>
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
