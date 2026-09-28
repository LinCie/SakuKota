import { ChevronDown, Search, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  data,
  useSearchParams,
  type ShouldRevalidateFunctionArgs,
} from "react-router";

import { categoryStyle } from "~/components/category-sign";
import { KecamatanBoard } from "~/components/kecamatan-board";
import { PlaceDetails, PlaceList } from "~/components/place-board";
import { SurvivalMap } from "~/components/survival-map";
import { getCategories, getPlaces } from "~/services/places";
import { getSubDistricts } from "~/services/subdistricts";
import type { Route } from "./+types/home";

export function meta() {
  return [
    { title: "Rantau — Peta bertahan hidup perantau Jakarta" },
    {
      name: "description",
      content:
        "Warung, penjahit, klinik, laundry, dan apotek terjangkau di sekitarmu, plus info banjir, keamanan, dan biaya harian per kecamatan.",
    },
  ];
}

function activeCategorySlugs(searchParams: URLSearchParams) {
  return (searchParams.get("kategori") ?? "").split(",").filter(Boolean);
}

// The kecamatan, categories, and search in the URL are applied by the
// backend; the selected place is picked from the loaded results.
export async function loader({ request }: Route.LoaderArgs) {
  const searchParams = new URL(request.url).searchParams;
  const [categories, subDistricts] = await Promise.all([
    getCategories(),
    getSubDistricts(),
  ]);
  if (!categories.ok) {
    throw data(categories.error.message, { status: categories.status });
  }
  if (!subDistricts.ok) {
    throw data(subDistricts.error.message, { status: subDistricts.status });
  }

  const subDistrict =
    subDistricts.data.find((sd) => sd.slug === searchParams.get("kecamatan")) ??
    subDistricts.data[0];
  const activeSlugs = activeCategorySlugs(searchParams);

  const places = subDistrict
    ? await getPlaces({
        subDistrictId: subDistrict.id,
        categoryIds: categories.data
          .filter((category) => activeSlugs.includes(category.slug))
          .map((category) => category.id),
        q: searchParams.get("q")?.trim(),
      })
    : null;
  if (places && !places.ok) {
    throw data(places.error.message, { status: places.status });
  }

  return {
    places: places ? places.data : [],
    categories: categories.data,
    subDistricts: subDistricts.data,
    subDistrict,
  };
}

// Selecting a place only changes `tempat`, which needs no fresh data.
export function shouldRevalidate({
  currentUrl,
  nextUrl,
  formMethod,
  defaultShouldRevalidate,
}: ShouldRevalidateFunctionArgs) {
  const withoutPlace = (url: URL) => {
    const params = new URLSearchParams(url.searchParams);
    params.delete("tempat");
    params.sort();
    return `${url.pathname}?${params}`;
  };
  if (!formMethod && withoutPlace(currentUrl) === withoutPlace(nextUrl)) {
    return false;
  }
  return defaultShouldRevalidate;
}

export default function Home({ loaderData }: Route.ComponentProps) {
  const {
    places: visiblePlaces,
    categories,
    subDistricts,
    subDistrict,
  } = loaderData;
  const [searchParams, setSearchParams] = useSearchParams();
  const [infoOpen, setInfoOpen] = useState(false);
  const sheetRef = useRef<HTMLDivElement>(null);

  const activeSlugs = activeCategorySlugs(searchParams);
  const query = searchParams.get("q") ?? "";
  const selectedParam = Number(searchParams.get("tempat"));

  const [draft, setDraft] = useState(query);

  function updateParams(changes: Record<string, string | null>) {
    setSearchParams(
      (previous) => {
        const next = new URLSearchParams(previous);
        for (const [key, value] of Object.entries(changes)) {
          if (value === null || value === "") next.delete(key);
          else next.set(key, value);
        }
        return next;
      },
      { replace: true, preventScrollReset: true },
    );
  }

  // Debounce typing into the URL.
  useEffect(() => {
    if (draft.trim() === query) return;
    const timeout = setTimeout(() => updateParams({ q: draft.trim() }), 250);
    return () => clearTimeout(timeout);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft, query]);

  const needle = query.trim();

  const selectedPlace =
    visiblePlaces.find((place) => place.id === selectedParam) ?? null;

  function selectPlace(placeId: number | null) {
    updateParams({ tempat: placeId === null ? null : String(placeId) });
  }

  function toggleCategory(slug: string) {
    const next = activeSlugs.includes(slug)
      ? activeSlugs.filter((s) => s !== slug)
      : [...activeSlugs, slug];
    const ordered = categories
      .map((category) => category.slug)
      .filter((s) => next.includes(s));
    updateParams({
      kategori: ordered.length === categories.length ? null : ordered.join(","),
    });
  }

  function clearSearch() {
    setDraft("");
    updateParams({ q: null });
  }

  // Escape closes the place details.
  useEffect(() => {
    if (!selectedPlace) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !infoOpen) selectPlace(null);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedPlace, infoOpen]);

  const allActive = activeSlugs.length === 0;
  const areaName = subDistrict?.name ?? "Jakarta";

  let emptyState: {
    message: string;
    action: string;
    onAction: () => void;
  } | null = null;
  if (visiblePlaces.length === 0) {
    emptyState = needle
      ? {
          message: `Tidak ada hasil untuk “${query}” di ${areaName}.`,
          action: "Hapus pencarian",
          onAction: clearSearch,
        }
      : allActive
        ? {
            message: `Belum ada tempat yang tercatat di ${areaName}. Coba kecamatan lain.`,
            action: "Lihat info kecamatan",
            onAction: () => setInfoOpen(true),
          }
        : {
            message: `Belum ada tempat untuk kategori ini di ${areaName}.`,
            action: "Tampilkan semua kategori",
            onAction: () => updateParams({ kategori: null }),
          };
  }

  return (
    <div className="flex min-h-dvh flex-col md:h-dvh">
      {/* Header board */}
      <header className="bg-board text-paint">
        <div className="flex flex-col gap-3 px-3 py-3 md:px-5 lg:flex-row lg:items-center lg:gap-6">
          <div className="flex items-center gap-4">
            <a
              href="/"
              className="bg-board-deep focus-visible:outline-sign shrink-0 rounded-[5px] p-1 focus-visible:outline-2 focus-visible:outline-offset-2"
            >
              <span className="border-sign font-sign text-sign block rounded-[3px] border-2 px-3 pt-1.5 pb-1 text-2xl leading-none tracking-wide">
                Rantau
              </span>
            </a>
            <p className="text-paint/85 max-w-[26ch] text-sm leading-snug">
              Peta bertahan hidup untuk perantau di Jakarta
            </p>
          </div>

          <div className="flex flex-1 flex-col gap-2 sm:flex-row sm:items-center lg:justify-end">
            <div className="flex gap-2">
              <label className="relative flex-1 sm:flex-none">
                <span className="sr-only">Kecamatan</span>
                <select
                  value={subDistrict?.slug ?? ""}
                  onChange={(event) =>
                    updateParams({
                      kecamatan: event.target.value,
                      tempat: null,
                    })
                  }
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
                onClick={() => setInfoOpen(true)}
                disabled={!subDistrict}
                className="border-ink bg-sign font-sign text-ink hover:bg-sign-soft focus-visible:outline-sign h-11 shrink-0 rounded-[3px] border-2 px-4 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-50"
              >
                Lihat info
              </button>
            </div>

            <form
              role="search"
              onSubmit={(event) => {
                event.preventDefault();
                updateParams({ q: draft.trim() });
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
                onChange={(event) => setDraft(event.target.value)}
                placeholder="Cari warung, apotek, nama tempat…"
                autoComplete="off"
                className="border-ink bg-paint text-ink placeholder:text-ink-soft focus-visible:border-sign h-11 w-full rounded-[3px] border-2 pr-11 pl-10 focus-visible:outline-none [&::-webkit-search-cancel-button]:appearance-none"
              />
              {draft ? (
                <button
                  type="button"
                  onClick={clearSearch}
                  aria-label="Hapus pencarian"
                  className="text-ink hover:bg-paint-deep focus-visible:bg-paint-deep focus-visible:outline-cat absolute top-1/2 right-1.5 grid size-8 -translate-y-1/2 place-items-center rounded-[3px] transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2"
                >
                  <X aria-hidden="true" className="size-4" strokeWidth={2.5} />
                </button>
              ) : null}
            </form>
          </div>
        </div>
        <div aria-hidden="true" className="bg-chili h-1.5" />
      </header>

      {/* Category signs */}
      <nav
        aria-label="Filter kategori"
        className="border-ink bg-paint flex flex-col gap-2 border-b-2 py-2.5 md:px-5 lg:flex-row lg:items-center lg:justify-between"
      >
        <div className="flex flex-wrap gap-2 px-3 md:px-0">
          <button
            type="button"
            aria-pressed={allActive}
            onClick={() => updateParams({ kategori: null })}
            className={`focus-visible:outline-cat h-10 shrink-0 rounded-[3px] border-2 px-3 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 ${
              allActive
                ? "border-ink bg-ink text-paint"
                : "border-ink/60 bg-paint text-ink hover:border-ink border-dashed"
            }`}
          >
            Semua
          </button>
          {categories.map((category) => {
            const style = categoryStyle(category);
            const active = activeSlugs.includes(category.slug);

            return (
              <button
                key={category.id}
                type="button"
                aria-pressed={active}
                onClick={() => toggleCategory(category.slug)}
                className={`focus-visible:outline-cat inline-flex h-10 shrink-0 items-center gap-2 rounded-[3px] border-2 pr-3 pl-1 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 ${
                  active
                    ? style.solid
                    : `bg-paint border-dashed ${style.outline} hover:border-solid`
                }`}
              >
                <span
                  aria-hidden="true"
                  className={`font-sign grid size-7 place-items-center rounded-[2px] text-sm leading-none ${
                    active ? "bg-paint/20" : `border-2 ${style.solid}`
                  }`}
                >
                  {style.code}
                </span>
                {style.label}
              </button>
            );
          })}
        </div>
        <p
          aria-live="polite"
          className="text-ink-soft shrink-0 px-3 text-sm md:px-0"
        >
          <span className="font-sign text-ink text-base tabular-nums">
            {visiblePlaces.length}
          </span>{" "}
          tempat di Kec. {areaName}
          {needle ? <> untuk “{query}”</> : null}
        </p>
      </nav>

      {/* Map + menu board */}
      <main className="flex flex-1 flex-col md:grid md:min-h-0 md:grid-cols-[2fr_1fr]">
        <section
          aria-label="Peta tempat penting"
          className="md:border-ink relative h-[58dvh] min-h-80 md:h-auto md:border-r-2"
        >
          <SurvivalMap
            places={visiblePlaces}
            focusPlaces={visiblePlaces}
            focusKey={subDistrict?.slug ?? ""}
            selectedId={selectedPlace?.id ?? null}
            onSelect={selectPlace}
            sheetRef={sheetRef}
          >
            {emptyState ? (
              <div className="bg-paint/70 absolute inset-0 z-1000 grid place-items-center p-4">
                <div className="border-ink bg-paint flex max-w-sm flex-col items-center gap-3 rounded-[5px] border-2 p-5 text-center shadow-[0_8px_24px_rgb(26_26_23/0.2)]">
                  <p className="text-ink font-semibold text-pretty">
                    {emptyState.message}
                  </p>
                  <button
                    type="button"
                    onClick={emptyState.onAction}
                    className="border-ink bg-sign font-sign text-ink hover:bg-sign-soft focus-visible:outline-cat h-10 rounded-[3px] border-2 px-4 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
                  >
                    {emptyState.action}
                  </button>
                </div>
              </div>
            ) : null}
          </SurvivalMap>
        </section>

        <aside
          aria-label="Daftar dan detail tempat"
          className="bg-paint md:min-h-0 md:overflow-y-auto"
        >
          <div className={selectedPlace ? "md:hidden" : undefined}>
            <h2 className="border-ink font-sign text-ink border-b-2 px-4 py-3 text-base md:px-5">
              Daftar tempat
            </h2>
            <PlaceList
              places={visiblePlaces}
              onSelect={selectPlace}
              emptyMessage={emptyState?.message ?? ""}
            />
          </div>

          {selectedPlace ? (
            <div
              ref={sheetRef}
              className="border-ink bg-paint fixed inset-x-0 bottom-0 z-1100 max-h-[60dvh] overflow-y-auto overscroll-contain rounded-t-[10px] border-2 border-b-0 shadow-[0_-10px_30px_rgb(26_26_23/0.3)] motion-safe:transition-transform motion-safe:duration-300 motion-safe:ease-out md:static md:z-auto md:max-h-none md:overflow-visible md:rounded-none md:border-0 md:shadow-none motion-safe:starting:translate-y-full md:motion-safe:starting:translate-y-0"
            >
              <PlaceDetails
                place={selectedPlace}
                onClose={() => selectPlace(null)}
              />
            </div>
          ) : null}
        </aside>
      </main>

      {subDistrict ? (
        <KecamatanBoard
          subDistrict={subDistrict}
          open={infoOpen}
          onClose={() => setInfoOpen(false)}
        />
      ) : null}
    </div>
  );
}

export function ErrorBoundary() {
  return (
    <main className="bg-board grid min-h-dvh place-items-center p-4">
      <div className="border-sign bg-board-deep text-paint flex max-w-md flex-col items-center gap-4 rounded-[5px] border-2 p-6 text-center">
        <span className="font-sign text-sign text-3xl">Rantau</span>
        <p className="font-semibold">Peta belum bisa dimuat.</p>
        <p className="text-paint/85 text-sm">
          Server Rantau sedang tidak merespons. Periksa koneksimu, lalu coba
          lagi.
        </p>
        <a
          href="/"
          className="border-ink bg-sign font-sign text-ink hover:bg-sign-soft focus-visible:outline-sign rounded-[3px] border-2 px-4 py-2 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
        >
          Coba lagi
        </a>
      </div>
    </main>
  );
}
