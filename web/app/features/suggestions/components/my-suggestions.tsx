import { Link } from "@tanstack/react-router";
import { MapPin } from "lucide-react";

import { CategoryBadge } from "~/features/places/components/category-sign";
import type { CategoryDetail } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { menuButtonClass } from "~/shared/components/menu-board";
import { MissingSlot } from "~/shared/components/missing-slot";
import { formatLocalDate } from "~/shared/format";

import type { Suggestion } from "../types";
import { SuggestionStatusBadge } from "./suggestion-status-badge";

type MySuggestionsProps = {
  /** The user's suggestions, newest first. */
  suggestions: Suggestion[];
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
};

/** The user's suggestions and where each stands in its review. */
export function MySuggestions({
  suggestions,
  categories,
  subDistricts,
}: MySuggestionsProps) {
  if (suggestions.length === 0) {
    return (
      <div>
        <MissingSlot>Belum ada usulan</MissingSlot>
      </div>
    );
  }

  return (
    <ul>
      {suggestions.map((suggestion) => {
        const category = categories.find(
          (c) => c.id === suggestion.category_id,
        );
        const subDistrict = subDistricts.find(
          (sd) => sd.id === suggestion.sub_district_id,
        );

        return (
          <li
            key={suggestion.id}
            className="border-ink/25 flex flex-col gap-1.5 border-t border-dashed py-3 first:border-t-0 first:pt-0"
          >
            <div className="flex items-start justify-between gap-3">
              <span className="text-ink font-semibold text-pretty">
                {suggestion.name}
              </span>
              <SuggestionStatusBadge status={suggestion.status} />
            </div>
            <div className="text-ink-soft flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
              {category ? <CategoryBadge category={category} /> : null}
              {subDistrict ? <span>Kec. {subDistrict.name}</span> : null}
            </div>
            {suggestion.status === "rejected" ? (
              <p className="text-ink text-sm text-pretty whitespace-pre-line">
                <span className="font-semibold">Alasan: </span>
                {suggestion.rejection_reason ?? "Tidak disebutkan."}
              </p>
            ) : null}
            {suggestion.status === "approved" &&
            suggestion.place_id !== null &&
            subDistrict ? (
              <Link
                to="/"
                search={{
                  kecamatan: subDistrict.slug,
                  tempat: suggestion.place_id,
                }}
                className={cn(
                  menuButtonClass,
                  "inline-flex items-center gap-1.5 self-start rounded-[3px] px-2 py-1 text-sm font-semibold",
                )}
              >
                <MapPin
                  aria-hidden="true"
                  className="size-4"
                  strokeWidth={2.5}
                />
                Lihat di peta
              </Link>
            ) : null}
            <time
              dateTime={suggestion.created_at}
              className="text-ink-soft text-sm"
            >
              Dikirim {formatLocalDate(new Date(suggestion.created_at))}
            </time>
          </li>
        );
      })}
    </ul>
  );
}
