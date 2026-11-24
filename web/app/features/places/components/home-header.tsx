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
import { cn } from "~/shared/cn";
import { SiteHeader } from "~/shared/components/site-header";

type HomeHeaderProps = {
  subDistricts: SubDistrict[];
  subDistrict: SubDistrict | undefined;
  /** The search segment of the bar. */
  search: ReactNode;
  /** The category filter, the bar's last segment. */
  filter: ReactNode;
  onKecamatanChange: (slug: string) => void;
  onOpenInfo: () => void;
};

// One pill holding segments: the pill draws the frame and, while its search
// input has focus, the focus ring.
const barClass =
  "border-input bg-card flex h-11 min-w-0 items-center rounded-full border transition-[border-color,box-shadow] has-[input:focus-visible]:border-ring has-[input:focus-visible]:ring-3 has-[input:focus-visible]:ring-ring/50 md:h-10";

const dividerClass = "bg-border h-5 w-px shrink-0";

/**
 * The map's header: the wordmark, one search bar (kecamatan and its info,
 * search, category filter), and the account. From 1024px it is one row; from
 * 640px the bar takes a second row; on phones the kecamatan joins the top row
 * as its own pill, search and filter fill the second, and the info moves onto
 * the map.
 */
export function HomeHeader({
  subDistricts,
  subDistrict,
  search,
  filter,
  onKecamatanChange,
  onOpenInfo,
}: HomeHeaderProps) {
  const items = subDistricts.map((sd) => ({
    value: sd.slug,
    label: `Kec. ${sd.name}`,
  }));

  return (
    <SiteHeader className="grid grid-cols-[auto_minmax(0,1fr)_auto] gap-2 sm:grid-cols-[minmax(0,1fr)_auto] sm:gap-x-3 sm:gap-y-2.5 lg:flex lg:justify-start lg:gap-8">
      {/* On phones the bar dissolves: its kecamatan pill and its search pill sit in the grid. */}
      <div
        className={cn(
          barClass,
          "contents sm:col-span-full sm:flex lg:max-w-3xl lg:flex-1",
        )}
      >
        <div className="border-input bg-card flex h-11 min-w-0 items-center rounded-full border sm:h-full sm:shrink-0 sm:border-0 sm:bg-transparent">
          <Select
            items={items}
            value={subDistrict?.slug ?? null}
            onValueChange={(slug) => {
              if (slug) onKecamatanChange(slug);
            }}
          >
            <SelectTrigger
              aria-label="Kecamatan"
              className="hover:bg-muted aria-expanded:bg-muted min-w-0 flex-1 gap-2 rounded-full border-0 bg-transparent pr-3 pl-3.5 font-semibold focus-visible:ring-3 data-[size=default]:h-full sm:max-w-64 sm:flex-none sm:rounded-r-none md:data-[size=default]:h-full"
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
            variant="ghost"
            size="icon-sm"
            onClick={onOpenInfo}
            disabled={!subDistrict}
            aria-label="Info kecamatan"
            title="Info kecamatan"
            className="text-primary hover:text-primary mr-1 hidden sm:inline-flex"
          >
            <Info aria-hidden="true" className="size-[18px]" />
          </Button>
        </div>

        <span
          aria-hidden="true"
          className={cn(dividerClass, "max-sm:hidden")}
        />

        <div className={cn(barClass, "col-span-full sm:contents")}>
          {search}
          <span aria-hidden="true" className={dividerClass} />
          {filter}
        </div>
      </div>

      <AccountControl className="col-start-3 row-start-1 sm:col-start-2 lg:ml-auto" />
    </SiteHeader>
  );
}
