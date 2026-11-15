import { ChevronDown } from "lucide-react";
import type { ReactNode } from "react";

import { AccountControl } from "~/features/auth/components/account-control";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { signButtonClass } from "~/shared/components/menu-board";
import { Wordmark } from "~/shared/components/wordmark";

type HomeHeaderProps = {
  subDistricts: SubDistrict[];
  subDistrict: SubDistrict | undefined;
  /** The search field; from 640px it sits in the header. */
  search: ReactNode;
  onKecamatanChange: (slug: string) => void;
  onOpenInfo: () => void;
};

/**
 * The top board: wordmark, kecamatan picker, search, and account. On phones it
 * is one row (wordmark, picker, account); search moves to the category strip
 * and "Lihat info" onto the map.
 */
export function HomeHeader({
  subDistricts,
  subDistrict,
  search,
  onKecamatanChange,
  onOpenInfo,
}: HomeHeaderProps) {
  return (
    <header className="bg-board text-paint">
      <div className="flex items-center gap-2 px-3 py-2 sm:grid sm:grid-cols-[minmax(0,1fr)_auto] sm:gap-3 md:px-5 md:py-3 lg:flex lg:gap-6">
        <Wordmark />

        <div className="flex min-w-0 flex-1 flex-col gap-2 sm:col-span-2 sm:flex-row sm:items-center lg:justify-end">
          <div className="flex gap-2">
            <label className="relative min-w-0 flex-1 sm:flex-none">
              <span className="sr-only">Kecamatan</span>
              <select
                value={subDistrict?.slug ?? ""}
                onChange={(event) => onKecamatanChange(event.target.value)}
                className="border-paint/35 bg-board-deep text-paint hover:border-paint/60 focus-visible:border-sign h-11 w-full appearance-none truncate rounded-[3px] border-2 py-0 pr-9 pl-3 font-semibold transition-colors focus-visible:outline-none sm:w-64 sm:pr-10"
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
                className="text-sign pointer-events-none absolute top-1/2 right-2.5 size-5 -translate-y-1/2 sm:right-3"
                strokeWidth={2.5}
              />
            </label>
            <button
              type="button"
              onClick={onOpenInfo}
              disabled={!subDistrict}
              className={cn(
                signButtonClass,
                "focus-visible:outline-sign hidden h-11 shrink-0 disabled:opacity-50 sm:block",
              )}
            >
              Lihat info
            </button>
          </div>

          <div className="hidden sm:block sm:flex-1 lg:max-w-md">{search}</div>
        </div>

        <AccountControl className="sm:col-start-2 sm:row-start-1 sm:justify-self-end" />
      </div>
      <div aria-hidden="true" className="bg-chili h-1.5" />
    </header>
  );
}
