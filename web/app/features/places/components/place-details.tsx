import { ArrowLeft, Star, X } from "lucide-react";
import { useEffect, useRef } from "react";

import { MediaGallery } from "~/shared/components/media-gallery";
import { DetailBlock, DetailRow } from "~/shared/components/menu-board";
import { MissingSlot, orMissing } from "~/shared/components/missing-slot";
import { Button } from "~/shared/components/ui/button";
import { formatNumber, formatRating } from "~/shared/format";
import { attributeValue } from "./attribute-value";
import type { CategoryField, Place } from "../types";
import { CategoryBadge } from "./category-sign";

type PlaceDetailsProps = {
  place: Place;
  /** Custom fields of the place's category, sorted by `sort_order`. */
  fields: CategoryField[];
  onClose: () => void;
};

/** One place's details, with a way back to the list. */
export function PlaceDetails({ place, fields, onClose }: PlaceDetailsProps) {
  const headingRef = useRef<HTMLHeadingElement>(null);

  // Short values sit on rows; free text follows as full-width blocks, so
  // `sort_order` applies within each group.
  const rowFields = fields.filter((field) => field.data_type !== "text");
  const textFields = fields.filter((field) => field.data_type === "text");

  useEffect(() => {
    headingRef.current?.focus({ preventScroll: true });
  }, [place.id]);

  return (
    <article aria-labelledby="place-name" className="flex flex-col">
      <div className="flex items-center justify-between gap-2 border-b px-2 py-2 md:px-3">
        <Button
          variant="ghost"
          onClick={onClose}
          className="hidden md:inline-flex"
        >
          <ArrowLeft aria-hidden="true" data-icon="inline-start" />
          Daftar tempat
        </Button>
        <span
          aria-hidden="true"
          className="bg-input mx-auto h-1.5 w-10 rounded-full md:hidden"
        />
        <Button
          variant="ghost"
          size="icon"
          onClick={onClose}
          aria-label="Tutup detail tempat"
          className="md:hidden"
        >
          <X aria-hidden="true" className="size-5" />
        </Button>
      </div>

      <div className="flex flex-col gap-4 px-4 pt-4 pb-2 md:px-5">
        <div>
          <CategoryBadge category={place.category} />
        </div>

        <h2
          id="place-name"
          ref={headingRef}
          tabIndex={-1}
          className="text-foreground -mt-1 text-xl leading-snug font-semibold tracking-[-0.01em] text-balance outline-none"
        >
          {place.name}
        </h2>

        <MediaGallery
          medias={place.medias}
          alt={`Foto ${place.name}`}
          missing="Foto tidak tersedia"
          aspect="aspect-4/3"
        />

        {place.short_description ? (
          <p className="text-foreground max-w-prose text-pretty">
            {place.short_description}
          </p>
        ) : (
          <div>
            <MissingSlot>Deskripsi tidak tersedia</MissingSlot>
          </div>
        )}

        <dl className="divide-y border-t">
          <DetailRow label="Rating">
            {place.rating_average === null ? (
              <MissingSlot>Belum ada ulasan</MissingSlot>
            ) : (
              <span className="inline-flex items-center gap-1.5">
                <Star
                  aria-hidden="true"
                  className="fill-kunyit text-kunyit size-4 shrink-0 self-center"
                />
                {formatRating(place.rating_average)}
                <span className="text-muted-foreground text-sm font-normal">
                  ({formatNumber(place.rating_count)} ulasan)
                </span>
              </span>
            )}
          </DetailRow>
          <DetailRow label="Jam buka">
            {orMissing(place.operating_hours, "Jam buka tidak tersedia")}
          </DetailRow>
          <DetailRow label="Kisaran harga">
            {orMissing(place.estimated_price_range, "Harga tidak tersedia")}
          </DetailRow>
          {rowFields.map((field) => (
            <DetailRow key={field.id} label={field.label}>
              {attributeValue(field, place)}
            </DetailRow>
          ))}
          {textFields.map((field) => (
            <DetailBlock key={field.id} label={field.label}>
              {attributeValue(field, place)}
            </DetailBlock>
          ))}
          <DetailBlock label="Alamat">
            {orMissing(place.address, "Alamat tidak tersedia")}
          </DetailBlock>
        </dl>
      </div>
    </article>
  );
}
