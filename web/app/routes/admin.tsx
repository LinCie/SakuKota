import { Tabs } from "@base-ui/react/tabs";
import {
  createFileRoute,
  Link,
  redirect,
  useRouter,
} from "@tanstack/react-router";
import { ArrowLeft, Check, Plus } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

import { AccountControl } from "~/features/auth/components/account-control";
import { getSession, isAdmin, useSession } from "~/features/auth/session";
import { getCategories, getPlaces } from "~/features/places/api";
import { CodePlate } from "~/features/places/components/category-sign";
import { PlaceSheet } from "~/features/places/components/place-sheet";
import type { Place } from "~/features/places/types";
import { getSubDistricts } from "~/features/subdistricts/api";
import { SubDistrictSheet } from "~/features/subdistricts/components/subdistrict-sheet";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { SelectControl } from "~/shared/components/form-fields";
import { MissingSlot } from "~/shared/components/missing-slot";
import { Wordmark } from "~/shared/components/wordmark";
import { formatNumber, formatRupiah } from "~/shared/format";

type AdminSearch = {
  /** Absent means the places tab. */
  tab?: "kecamatan";
};

export const Route = createFileRoute("/admin")({
  validateSearch: (search: Record<string, unknown>): AdminSearch => ({
    tab: search.tab === "kecamatan" ? "kecamatan" : undefined,
  }),
  beforeLoad: ({ location }) => {
    if (!getSession()) {
      throw redirect({
        to: "/masuk",
        search: { kembali: location.href },
        replace: true,
      });
    }
  },
  loader: async () => {
    const [categories, subDistricts, places] = await Promise.all([
      getCategories(),
      getSubDistricts(),
      getPlaces(),
    ]);
    if (!categories.ok) throw new Error(categories.error.message);
    if (!subDistricts.ok) throw new Error(subDistricts.error.message);
    if (!places.ok) throw new Error(places.error.message);

    return {
      categories: categories.data,
      subDistricts: subDistricts.data,
      places: places.data,
    };
  },
  head: () => ({ meta: [{ title: "Admin · Rantau" }] }),
  component: Admin,
  errorComponent: AdminError,
});

// -----------------------------------------------------------------------------
// Page
// -----------------------------------------------------------------------------

function AdminHeader() {
  return (
    <header className="bg-board text-paint">
      <div className="flex items-center justify-between gap-3 px-3 py-3 md:px-5">
        <div className="flex min-w-0 items-center gap-3">
          <Wordmark tagline={false} />
          <span className="bg-ink font-sign text-sign rounded-[3px] px-2 pt-1.5 pb-1 text-sm leading-none">
            Admin
          </span>
        </div>
        <AccountControl className="shrink-0" />
      </div>
      <div aria-hidden="true" className="bg-chili h-1.5" />
    </header>
  );
}

const quietLink =
  "text-ink hover:bg-sign-soft focus-visible:bg-sign-soft focus-visible:outline-cat inline-flex h-10 items-center gap-2 rounded-[3px] px-2 text-sm font-semibold transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2";

function BackToMap() {
  return (
    <Link to="/" className={quietLink}>
      <ArrowLeft aria-hidden="true" className="size-4" strokeWidth={2.5} />
      Kembali ke peta
    </Link>
  );
}

function Admin() {
  const session = useSession();
  const { categories, subDistricts, places } = Route.useLoaderData();
  const { tab } = Route.useSearch();
  const navigate = Route.useNavigate();
  const router = useRouter();

  const [placeOpen, setPlaceOpen] = useState(false);
  const [subDistrictOpen, setSubDistrictOpen] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [highlight, setHighlight] = useState<string | null>(null);

  // The notice and the new row's highlight fade after a moment.
  useEffect(() => {
    if (!notice) return;
    const timeout = setTimeout(() => setNotice(null), 6000);
    return () => clearTimeout(timeout);
  }, [notice]);
  useEffect(() => {
    if (!highlight) return;
    const timeout = setTimeout(() => setHighlight(null), 2500);
    return () => clearTimeout(timeout);
  }, [highlight]);

  if (!isAdmin(session)) {
    return <AdminGate signedIn={session !== null} />;
  }

  async function placeCreated(place: Place) {
    await router.invalidate();
    setNotice(`Tempat ditambahkan: ${place.name}`);
    setHighlight(`tempat-${place.id}`);
  }

  async function subDistrictCreated(subDistrict: SubDistrict) {
    await router.invalidate();
    setNotice(`Kecamatan ditambahkan: ${subDistrict.name}`);
    setHighlight(`kecamatan-${subDistrict.id}`);
  }

  return (
    <div className="bg-paint text-ink flex min-h-dvh flex-col">
      <AdminHeader />

      <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col">
        <h1 className="sr-only">Admin Rantau</h1>

        <div className="px-1 pt-2 md:px-3">
          <BackToMap />
        </div>

        <Tabs.Root
          value={tab ?? "tempat"}
          onValueChange={(value) =>
            void navigate({
              search: { tab: value === "kecamatan" ? "kecamatan" : undefined },
              replace: true,
              resetScroll: false,
            })
          }
          className="flex flex-1 flex-col"
        >
          <Tabs.List
            aria-label="Jenis data"
            className="grid grid-cols-2 gap-2 px-3 pt-1 md:px-5"
          >
            <Tab value="tempat">Tempat</Tab>
            <Tab value="kecamatan">Kecamatan</Tab>
          </Tabs.List>

          <div role="status" className="px-3 pt-3 empty:hidden md:px-5">
            {notice ? (
              <p className="border-ink bg-sign-soft flex items-center gap-2 rounded-[3px] border-2 px-3 py-2.5 text-sm font-semibold">
                <Check
                  aria-hidden="true"
                  className="size-4 shrink-0"
                  strokeWidth={2.5}
                />
                {notice}
              </p>
            ) : null}
          </div>

          <Tabs.Panel
            value="tempat"
            className="flex flex-1 flex-col outline-none"
          >
            <PlacesPanel
              places={places}
              subDistricts={subDistricts}
              highlightId={highlight}
              onAdd={() => setPlaceOpen(true)}
            />
          </Tabs.Panel>
          <Tabs.Panel
            value="kecamatan"
            className="flex flex-1 flex-col outline-none"
          >
            <SubDistrictsPanel
              places={places}
              subDistricts={subDistricts}
              highlightId={highlight}
              onAdd={() => setSubDistrictOpen(true)}
            />
          </Tabs.Panel>
        </Tabs.Root>
      </main>

      <PlaceSheet
        open={placeOpen}
        onOpenChange={setPlaceOpen}
        categories={categories}
        subDistricts={subDistricts}
        places={places}
        onCreated={placeCreated}
      />
      <SubDistrictSheet
        open={subDistrictOpen}
        onOpenChange={setSubDistrictOpen}
        onCreated={subDistrictCreated}
      />
    </div>
  );
}

function Tab({ value, children }: { value: string; children: ReactNode }) {
  return (
    <Tabs.Tab
      value={value}
      className="font-sign text-ink border-ink/60 bg-paint hover:border-ink focus-visible:outline-cat data-active:border-ink data-active:bg-sign grid h-12 cursor-pointer place-items-center rounded-[3px] border-2 border-dashed text-lg leading-none transition-colors hover:border-solid focus-visible:outline-2 focus-visible:outline-offset-2 data-active:border-solid"
    >
      {children}
    </Tabs.Tab>
  );
}

// -----------------------------------------------------------------------------
// Panels
// -----------------------------------------------------------------------------

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
        "border-ink bg-sign font-sign text-ink hover:bg-sign-soft focus-visible:outline-cat inline-flex h-11 items-center justify-center gap-2 rounded-[3px] border-2 px-4 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-50",
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

function PlacesPanel({
  places,
  subDistricts,
  highlightId,
  onAdd,
}: {
  places: Place[];
  subDistricts: SubDistrict[];
  highlightId: string | null;
  onAdd: () => void;
}) {
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

function SubDistrictsPanel({
  places,
  subDistricts,
  highlightId,
  onAdd,
}: {
  places: Place[];
  subDistricts: SubDistrict[];
  highlightId: string | null;
  onAdd: () => void;
}) {
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

// -----------------------------------------------------------------------------
// Gate and error
// -----------------------------------------------------------------------------

function AdminGate({ signedIn }: { signedIn: boolean }) {
  return (
    <div className="bg-paint text-ink flex min-h-dvh flex-col">
      <AdminHeader />
      <main className="flex flex-1 flex-col items-center px-3 pt-6 pb-10">
        <div className="border-ink bg-paint w-full max-w-104 rounded-[5px] border-2 p-1">
          <div className="border-ink/40 flex flex-col gap-4 rounded-[3px] border p-4 sm:p-5">
            <h1 className="font-sign text-xl leading-tight">
              {signedIn ? "Khusus admin" : "Masuk dulu"}
            </h1>
            <p className="text-pretty">
              {signedIn
                ? "Akun ini belum punya akses admin. Masuk dengan akun admin untuk menambah tempat dan kecamatan."
                : "Masuk dengan akun admin untuk menambah tempat dan kecamatan."}
            </p>
            {signedIn ? null : (
              <Link
                to="/masuk"
                search={{ kembali: "/admin" }}
                className="border-ink bg-sign font-sign text-ink hover:bg-sign-soft focus-visible:outline-cat grid h-11 place-items-center rounded-[3px] border-2 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
              >
                Masuk
              </Link>
            )}
            <BackToMap />
          </div>
        </div>
      </main>
    </div>
  );
}

function AdminError() {
  return (
    <main className="bg-board grid min-h-dvh place-items-center p-4">
      <div className="border-sign bg-board-deep text-paint flex max-w-md flex-col items-center gap-4 rounded-[5px] border-2 p-6 text-center">
        <span className="font-sign text-sign text-3xl">Rantau</span>
        <p className="font-semibold">Halaman admin belum bisa dimuat.</p>
        <p className="text-paint/85 text-sm">
          Server Rantau sedang tidak merespons. Periksa koneksimu, lalu coba
          lagi.
        </p>
        <a
          href="/admin"
          className="border-ink bg-sign font-sign text-ink hover:bg-sign-soft focus-visible:outline-sign rounded-[3px] border-2 px-4 py-2 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
        >
          Coba lagi
        </a>
      </div>
    </main>
  );
}
