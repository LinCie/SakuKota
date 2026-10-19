import { Plus } from "lucide-react";
import { useState, type ReactNode } from "react";

import { CodePlate } from "~/features/places/components/category-sign";
import type { Place } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { SelectControl } from "~/shared/components/form-fields";
import { MissingSlot } from "~/shared/components/missing-slot";
import { signButtonClass } from "~/shared/components/menu-board";
import { formatNumber, formatRupiah } from "~/shared/format";

function AddPlate({
  onClick,
  disabled,
  className,
  children,
}: {
  onClick: () => void;
  disabled?: boolean;
  className?: string;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        signButtonClass,
        "inline-flex h-11 items-center justify-center gap-2 disabled:opacity-50",
        className,
      )}
    >
      <Plus aria-hidden="true" className="size-4" strokeWidth={3} />
      {children}
    </button>
  );
}

/** The add plate pinned to the bottom of a phone screen, within thumb reach. */
function StickyAdd({
  onClick,
  disabled,
  children,
}: {
  onClick: () => void;
  disabled?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="border-ink bg-paint sticky bottom-0 z-10 mt-auto border-t-2 p-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] md:hidden">
      <AddPlate onClick={onClick} disabled={disabled} className="w-full">
        {children}
      </AddPlate>
    </div>
  );
}

const rowClass =
  "border-ink/20 flex items-start gap-3 border-b border-dashed px-3 py-3 transition-colors duration-700 last:border-b-0 md:px-5";

function Count({ n, children }: { n: number; children: ReactNode }) {
  return (
    <p className="text-ink-soft px-3 pb-3 text-sm md:px-5">
      <span className="font-sign text-ink tabular-nums">{formatNumber(n)}</span>{" "}
      {children}
    </p>
  );
}

type PanelProps = {
  places: Place[];
  subDistricts: SubDistrict[];
  highlightId: string | null;
  onAdd: () => void;
};

/** The admin's list of places, filterable by kecamatan. */
export function PlacesPanel({
  places,
  subDistricts,
  highlightId,
  onAdd,
}: PanelProps) {
  const [filter, setFilter] = useState("");
  const names = new Map(subDistricts.map((sd) => [sd.id, sd.name]));
  const rows = places
    .filter((place) => !filter || String(place.sub_district_id) === filter)
    .sort((a, b) => b.id - a.id);
  const area = filter ? names.get(Number(filter)) : undefined;
  const noSubDistricts = subDistricts.length === 0;

  return (
    <>
      <div className="flex items-center gap-3 px-3 py-3 md:px-5">
        <div className="flex-1 md:max-w-64">
          <label htmlFor="filter-kecamatan" className="sr-only">
            Saring menurut kecamatan
          </label>
          <SelectControl
            id="filter-kecamatan"
            value={filter}
            onValueChange={setFilter}
            placeholder="Semua kecamatan"
            options={subDistricts.map((sd) => ({
              value: String(sd.id),
              label: `Kec. ${sd.name}`,
            }))}
          />
        </div>
        <AddPlate
          onClick={onAdd}
          disabled={noSubDistricts}
          className="ml-auto hidden md:inline-flex"
        >
          Tambah tempat
        </AddPlate>
      </div>

      <div aria-live="polite">
        <Count n={rows.length}>tempat{area ? ` di Kec. ${area}` : ""}</Count>
      </div>

      {rows.length === 0 ? (
        <p className="border-ink/20 text-ink-soft border-t border-dashed px-3 py-6 text-sm md:px-5">
          {noSubDistricts
            ? "Belum ada kecamatan. Tambah kecamatan dulu, baru tempatnya."
            : area
              ? `Belum ada tempat di Kec. ${area}.`
              : "Belum ada tempat yang tercatat."}
        </p>
      ) : (
        <ul className="border-ink/20 border-t border-dashed">
          {rows.map((place) => (
            <li
              key={place.id}
              className={cn(
                rowClass,
                highlightId === `tempat-${place.id}` && "bg-sign-soft",
              )}
            >
              <CodePlate category={place.category} className="mt-0.5" />
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="font-semibold">{place.name}</span>
                <span className="text-ink-soft truncate text-sm">
                  {place.address ?? "Alamat tidak tersedia"}
                </span>
                <span className="text-ink-soft mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
                  Kec. {names.get(place.sub_district_id) ?? "?"}
                  {place.location ? null : (
                    <MissingSlot>Tanpa lokasi</MissingSlot>
                  )}
                </span>
              </div>
              {place.estimated_price_range ? (
                <span className="mt-0.5 shrink-0 text-right text-sm font-semibold tabular-nums">
                  {place.estimated_price_range}
                </span>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      <StickyAdd onClick={onAdd} disabled={noSubDistricts}>
        Tambah tempat
      </StickyAdd>
    </>
  );
}

/** The admin's list of kecamatan, with how complete each one's info is. */
export function SubDistrictsPanel({
  places,
  subDistricts,
  highlightId,
  onAdd,
}: PanelProps) {
  const rows = [...subDistricts].sort((a, b) => b.id - a.id);

  return (
    <>
      <div className="flex items-center justify-between gap-3 px-3 py-3 md:px-5">
        <p aria-live="polite" className="text-ink-soft text-sm">
          <span className="font-sign text-ink tabular-nums">
            {formatNumber(rows.length)}
          </span>{" "}
          kecamatan
        </p>
        <AddPlate onClick={onAdd} className="hidden md:inline-flex">
          Tambah kecamatan
        </AddPlate>
      </div>

      {rows.length === 0 ? (
        <p className="border-ink/20 text-ink-soft border-t border-dashed px-3 py-6 text-sm md:px-5">
          Belum ada kecamatan yang tercatat.
        </p>
      ) : (
        <ul className="border-ink/20 border-t border-dashed">
          {rows.map((sd) => {
            const missing = [
              sd.flood_history ? null : "banjir",
              sd.safety_crime_information ? null : "keamanan",
              sd.static_map_image_url ? null : "peta",
            ].filter((label) => label !== null);
            const placeCount = places.filter(
              (place) => place.sub_district_id === sd.id,
            ).length;

            return (
              <li
                key={sd.id}
                className={cn(
                  rowClass,
                  highlightId === `kecamatan-${sd.id}` && "bg-sign-soft",
                )}
              >
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <span className="font-semibold">Kec. {sd.name}</span>
                  <span className="text-ink-soft truncate text-sm">
                    {sd.slug} ·{" "}
                    <span className="tabular-nums">
                      {formatNumber(placeCount)}
                    </span>{" "}
                    tempat
                  </span>
                  {missing.length > 0 ? (
                    <span className="mt-0.5">
                      <MissingSlot>
                        Belum ada info {missing.join(", ")}
                      </MissingSlot>
                    </span>
                  ) : null}
                </div>
                <span className="mt-0.5 shrink-0 text-right text-sm font-semibold tabular-nums">
                  {sd.estimated_daily_spending !== null ? (
                    `${formatRupiah(sd.estimated_daily_spending)}/hari`
                  ) : (
                    <MissingSlot>Biaya belum diisi</MissingSlot>
                  )}
                </span>
              </li>
            );
          })}
        </ul>
      )}

      <StickyAdd onClick={onAdd}>Tambah kecamatan</StickyAdd>
    </>
  );
}
