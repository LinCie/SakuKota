import { Link } from "@tanstack/react-router";
import { MapPin } from "lucide-react";

import { CategoryBadge } from "~/features/places/components/category-sign";
import type { CategoryDetail } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { MissingSlot } from "~/shared/components/missing-slot";
import { buttonVariants } from "~/shared/components/ui/button";
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
            className="flex flex-col gap-1.5 border-t py-3 first:border-t-0 first:pt-0 last:pb-0"
          >
            <div className="flex items-start justify-between gap-3">
              <span className="text-foreground font-semibold text-pretty">
                {suggestion.name}
              </span>
              <SuggestionStatusBadge status={suggestion.status} />
            </div>
            <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
              {category ? <CategoryBadge category={category} /> : null}
              {subDistrict ? <span>Kec. {subDistrict.name}</span> : null}
            </div>
            {suggestion.status === "rejected" ? (
              <p className="text-foreground text-sm text-pretty whitespace-pre-line">
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
                  buttonVariants({ variant: "ghost", size: "sm" }),
                  "text-primary hover:text-primary -ml-2.5 self-start",
                )}
              >
                <MapPin aria-hidden="true" data-icon="inline-start" />
                Lihat di peta
              </Link>
            ) : null}
            <time
              dateTime={suggestion.created_at}
              className="text-muted-foreground text-xs"
            >
              Dikirim {formatLocalDate(new Date(suggestion.created_at))}
            </time>
          </li>
        );
      })}
    </ul>
  );
}
