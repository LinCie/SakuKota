import { Info, MapPin } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "~/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/shared/components/ui/select";
import { AccountControl } from "~/features/auth/components/account-control";
import type { SubDistrict } from "~/features/subdistricts/types";
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
 * The header: wordmark, kecamatan picker, search, and account. On phones it is
 * one row (wordmark, picker, account); search moves to the filter row and
 * "Lihat info" onto the map.
 */
export function HomeHeader({
  subDistricts,
  subDistrict,
  search,
  onKecamatanChange,
  onOpenInfo,
}: HomeHeaderProps) {
  const items = subDistricts.map((sd) => ({
    value: sd.slug,
    label: `Kec. ${sd.name}`,
  }));

  return (
    <header className="bg-background border-b">
      <div className="flex items-center gap-2 px-3 py-2 sm:grid sm:grid-cols-[minmax(0,1fr)_auto] sm:gap-3 md:px-5 md:py-3 lg:flex lg:gap-6">
        <Wordmark />

        <div className="flex min-w-0 flex-1 flex-col gap-2 sm:col-span-2 sm:flex-row sm:items-center lg:justify-end">
          <div className="flex gap-2">
            <Select
              items={items}
              value={subDistrict?.slug ?? null}
              onValueChange={(slug) => {
                if (slug) onKecamatanChange(slug);
              }}
            >
              <SelectTrigger
                aria-label="Kecamatan"
                className="min-w-0 flex-1 pr-3 pl-3.5 font-semibold sm:w-64 sm:flex-none"
              >
                <MapPin aria-hidden="true" className="text-primary" />
                <SelectValue className="truncate" />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false} align="start">
                {items.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              onClick={onOpenInfo}
              disabled={!subDistrict}
              className="hidden sm:inline-flex"
            >
              <Info aria-hidden="true" data-icon="inline-start" />
              Lihat info
            </Button>
          </div>

          <div className="hidden sm:block sm:flex-1 lg:max-w-md">{search}</div>
        </div>

        <AccountControl className="sm:col-start-2 sm:row-start-1 sm:justify-self-end" />
      </div>
    </header>
  );
}
