import { useNavigate } from "@tanstack/react-router";

import type { CategoryDetail } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { SelectControl } from "~/shared/components/field";
import { PageHeader } from "~/shared/components/page-header";
import { SearchField } from "~/shared/components/search-field";
import { formatNumber } from "~/shared/format";
import { useTableQuery } from "~/shared/hooks";

import { STATUS_LABELS } from "../components/suggestion-status-badge";
import { SuggestionsTable } from "../components/suggestions-table";
import {
  SUGGESTION_STATUSES,
  type Suggestion,
  type SuggestionStatus,
} from "../types";

type SuggestionsPageProps = {
  /** Oldest first. */
  suggestions: Suggestion[];
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  /** The status the list is filtered to; absent shows every status. */
  status: SuggestionStatus | undefined;
  onStatusChange: (status: SuggestionStatus | undefined) => void;
};

function suggestionSearchText(suggestion: Suggestion) {
  return [suggestion.name, suggestion.address ?? "", suggestion.user.name].join(
    " ",
  );
}

/** The suggestions list: a status filter, search, and a table to open one for review. */
export function SuggestionsPage({
  suggestions,
  categories,
  subDistricts,
  status,
  onStatusChange,
}: SuggestionsPageProps) {
  const navigate = useNavigate();
  const search = useTableQuery(suggestions, suggestionSearchText);

  return (
    <>
      <PageHeader
        title="Usulan"
        description="Tempat yang diusulkan pengguna. Yang disetujui masuk ke peta."
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex w-full max-w-xl flex-wrap items-center gap-3 sm:flex-nowrap">
          <SearchField
            value={search.query}
            onValueChange={search.setQuery}
            label="Cari usulan"
            placeholder="Cari nama, alamat, atau pengusul"
          />
          <div className="w-full sm:w-44 sm:shrink-0">
            <SelectControl
              aria-label="Status"
              value={status ?? ""}
              onValueChange={(value) =>
                onStatusChange(
                  SUGGESTION_STATUSES.find((s) => s === value) ?? undefined,
                )
              }
              placeholder="Semua status"
              options={SUGGESTION_STATUSES.map((s) => ({
                value: s,
                label: STATUS_LABELS[s],
              }))}
              className="border-ink/20 h-9 border text-sm"
            />
          </div>
        </div>
        <p className="text-ink-soft text-sm tabular-nums">
          {formatNumber(search.rows.length)} dari{" "}
          {formatNumber(suggestions.length)} usulan
        </p>
      </div>

      <SuggestionsTable
        suggestions={search.rows}
        categories={categories}
        subDistricts={subDistricts}
        onView={(suggestion) =>
          void navigate({
            to: "/admin/suggestions/$suggestionId",
            params: { suggestionId: String(suggestion.id) },
          })
        }
      />
    </>
  );
}
