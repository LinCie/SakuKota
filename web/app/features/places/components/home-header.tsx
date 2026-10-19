import { ChevronDown, Search, X } from "lucide-react";

import { AccountControl } from "~/features/auth/components/account-control";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { signButtonClass } from "~/shared/components/menu-board";
import { Wordmark } from "~/shared/components/wordmark";

type HomeHeaderProps = {
  subDistricts: SubDistrict[];
  subDistrict: SubDistrict | undefined;
  /** The search text as typed, before it is debounced into the URL. */
  draft: string;
  onDraftChange: (draft: string) => void;
  onSearch: () => void;
  onClearSearch: () => void;
  onKecamatanChange: (slug: string) => void;
  onOpenInfo: () => void;
};

/** The top board: wordmark, kecamatan picker, search, and account. */
export function HomeHeader({
  subDistricts,
  subDistrict,
  draft,
  onDraftChange,
  onSearch,
  onClearSearch,
  onKecamatanChange,
  onOpenInfo,
}: HomeHeaderProps) {
  return (
    <header className="bg-board text-paint">
      <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-3 md:px-5 lg:flex lg:gap-6">
        <Wordmark />

        <div className="col-span-2 flex flex-1 flex-col gap-2 sm:flex-row sm:items-center lg:justify-end">
          <div className="flex gap-2">
            <label className="relative flex-1 sm:flex-none">
              <span className="sr-only">Kecamatan</span>
              <select
                value={subDistrict?.slug ?? ""}
                onChange={(event) => onKecamatanChange(event.target.value)}
                className="border-paint/35 bg-board-deep text-paint hover:border-paint/60 focus-visible:border-sign h-11 w-full appearance-none rounded-[3px] border-2 py-0 pr-10 pl-3 font-semibold transition-colors focus-visible:outline-none sm:w-64"
              >
                {subDistricts.map((sd) => (
                  <option
                    key={sd.id}
                    value={sd.slug}
                    className="bg-paint text-ink"
                  >
                    Kec. {sd.name}
                  </option>
                ))}
              </select>
              <ChevronDown
                aria-hidden="true"
                className="text-sign pointer-events-none absolute top-1/2 right-3 size-5 -translate-y-1/2"
                strokeWidth={2.5}
              />
            </label>
            <button
              type="button"
              onClick={onOpenInfo}
              disabled={!subDistrict}
              className={cn(
                signButtonClass,
                "focus-visible:outline-sign h-11 shrink-0 disabled:opacity-50",
              )}
            >
              Lihat info
            </button>
          </div>

          <form
            role="search"
            onSubmit={(event) => {
              event.preventDefault();
              onSearch();
            }}
            className="relative sm:flex-1 lg:max-w-md"
          >
            <label htmlFor="search" className="sr-only">
              Cari tempat
            </label>
            <Search
              aria-hidden="true"
              className="text-ink-soft pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2"
              strokeWidth={2.5}
            />
            <input
              id="search"
              type="search"
              value={draft}
              onChange={(event) => onDraftChange(event.target.value)}
              placeholder="Cari warung, apotek, nama tempat…"
              autoComplete="off"
              className="border-ink bg-paint text-ink placeholder:text-ink-soft focus-visible:border-sign h-11 w-full rounded-[3px] border-2 pr-11 pl-10 focus-visible:outline-none [&::-webkit-search-cancel-button]:appearance-none"
            />
            {draft ? (
              <button
                type="button"
                onClick={onClearSearch}
                aria-label="Hapus pencarian"
                className="text-ink hover:bg-paint-deep focus-visible:bg-paint-deep focus-visible:outline-cat absolute top-1/2 right-1.5 grid size-8 -translate-y-1/2 place-items-center rounded-[3px] transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2"
              >
                <X aria-hidden="true" className="size-4" strokeWidth={2.5} />
              </button>
            ) : null}
          </form>
        </div>

        <AccountControl className="col-start-2 row-start-1 justify-self-end" />
      </div>
      <div aria-hidden="true" className="bg-chili h-1.5" />
    </header>
  );
}
