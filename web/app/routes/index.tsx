import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useEffectEvent, useRef, useState } from "react";

import type { GetPlacesParams } from "~/features/places/api";
import { CategoryNav } from "~/features/places/components/category-nav";
import { HomeHeader } from "~/features/places/components/home-header";
import {
  MapEmptyState,
  type EmptyState,
} from "~/features/places/components/map-empty-state";
import { PlaceDetails } from "~/features/places/components/place-details";
import { PlaceList } from "~/features/places/components/place-list";
import { SurvivalMap } from "~/features/places/components/survival-map";
import {
  categoriesQuery,
  placeQuery,
  placesQuery,
  useCategories,
  usePlaces,
} from "~/features/places/queries";
import type { CategoryDetail } from "~/features/places/types";
import {
  subDistrictsQuery,
  useSubDistricts,
} from "~/features/subdistricts/queries";
import type { SubDistrict } from "~/features/subdistricts/types";
import { KecamatanBoard } from "~/features/subdistricts/components/kecamatan-board";
import { ErrorBoard } from "~/shared/components/error-board";

type HomeSearch = {
  kecamatan?: string;
  /** Comma-separated category slugs; absent means every category. */
  kategori?: string;
  q?: string;
  /** Caps the places to the lowest ids while the map is zoomed out. */
  limit?: number;
  tempat?: number;
};

// Search params arrive JSON-parsed (`?q=2024` is a number). Malformed values
// are dropped instead of erroring.
function searchString(value: unknown) {
  if (typeof value === "number") return String(value);
  return typeof value === "string" && value !== "" ? value : undefined;
}

function activeCategorySlugs(kategori: string | undefined) {
  return (kategori ?? "").split(",").filter(Boolean);
}

// A limit must be a positive whole number; anything else is dropped.
function searchLimit(value: unknown) {
  return typeof value === "number" && Number.isInteger(value) && value > 0
    ? value
    : undefined;
}

type PlacesFilters = Pick<HomeSearch, "kecamatan" | "kategori" | "q" | "limit">;

/**
 * The kecamatan on show (the one in the URL, else the first) and the places
 * query for the filters. With no kecamatan at all there are no places either.
 */
function placesSelection(
  { kecamatan, kategori, q, limit }: PlacesFilters,
  categories: CategoryDetail[],
  subDistricts: SubDistrict[],
) {
  const subDistrict =
    subDistricts.find((sd) => sd.slug === kecamatan) ?? subDistricts[0];
  const activeSlugs = activeCategorySlugs(kategori);
  const params: GetPlacesParams = {
    subDistrictId: subDistrict?.id,
    categoryIds: categories
      .filter((category) => activeSlugs.includes(category.slug))
      .map((category) => category.id),
    q: q?.trim(),
    limit,
  };
  return { subDistrict, params };
}

export const Route = createFileRoute("/")({
  validateSearch: (search: Record<string, unknown>): HomeSearch => ({
    kecamatan: searchString(search.kecamatan),
    kategori: searchString(search.kategori),
    q: searchString(search.q),
    limit: searchLimit(search.limit),
    tempat: typeof search.tempat === "number" ? search.tempat : undefined,
  }),
  // The kecamatan, categories, and search are applied by the backend; the
  // selected place (`tempat`) is picked from the loaded results, so changing
  // it doesn't reload.
  loaderDeps: ({ search: { kecamatan, kategori, q, limit } }) => ({
    kecamatan,
    kategori,
    q,
    limit,
  }),
  loader: async ({ context: { queryClient }, deps }) => {
    const [categories, subDistricts] = await Promise.all([
      queryClient.query({ ...categoriesQuery(), staleTime: "static" }),
      queryClient.query({ ...subDistrictsQuery(), staleTime: "static" }),
    ]);
    const { params } = placesSelection(deps, categories, subDistricts);
    await queryClient.query({ ...placesQuery(params), staleTime: "static" });
  },
  head: () => ({
    meta: [
      { title: "Rantau — Peta bertahan hidup perantau Jakarta" },
      {
        name: "description",
        content:
          "Warung, penjahit, klinik, laundry, dan apotek terjangkau di sekitarmu, plus info banjir, keamanan, dan biaya harian per kecamatan.",
      },
    ],
  }),
  component: Home,
  errorComponent: HomeError,
});

function Home() {
  const search = Route.useSearch();
  const { data: categories } = useCategories();
  const { data: subDistricts } = useSubDistricts();
  const { subDistrict, params } = placesSelection(
    search,
    categories,
    subDistricts,
  );
  const { data: visiblePlaces } = usePlaces(params);
  const navigate = Route.useNavigate();
  const [infoOpen, setInfoOpen] = useState(false);
  const sheetRef = useRef<HTMLDivElement>(null);

  const activeSlugs = activeCategorySlugs(search.kategori);
  const query = search.q ?? "";

  const [draft, setDraft] = useState(query);

  function updateParams(changes: HomeSearch) {
    void navigate({
      search: (previous) => ({ ...previous, ...changes }),
      replace: true,
      resetScroll: false,
    });
  }

  // Debounce typing into the URL.
  const applyDraft = useEffectEvent(() =>
    updateParams({ q: draft.trim() || undefined }),
  );
  useEffect(() => {
    if (draft.trim() === query) return;
    const timeout = setTimeout(applyDraft, 250);
    return () => clearTimeout(timeout);
  }, [draft, query]);

  const needle = query.trim();

  const listedPlace =
    visiblePlaces.find((place) => place.id === search.tempat) ?? null;
  // A zoomed-out list can drop the selected place; it is fetched so its details stay open.
  const { data: detachedPlace } = useQuery({
    ...placeQuery(search.tempat ?? 0),
    enabled: search.tempat !== undefined && listedPlace === null,
  });
  const selectedPlace = listedPlace ?? detachedPlace ?? null;
  const selectedFields =
    categories.find((category) => category.id === selectedPlace?.category.id)
      ?.fields ?? [];

  function selectPlace(placeId: number | null) {
    updateParams({ tempat: placeId ?? undefined });
  }

  function setLimit(limit: number | undefined) {
    if (limit !== search.limit) updateParams({ limit });
  }

  function toggleCategory(slug: string) {
    const next = activeSlugs.includes(slug)
      ? activeSlugs.filter((s) => s !== slug)
      : [...activeSlugs, slug];
    const ordered = categories
      .map((category) => category.slug)
      .filter((s) => next.includes(s));
    updateParams({
      kategori:
        ordered.length === categories.length ? undefined : ordered.join(","),
    });
  }

  function clearSearch() {
    setDraft("");
    updateParams({ q: undefined });
  }

  // Escape closes the place details.
  const closeOnEscape = useEffectEvent((event: KeyboardEvent) => {
    if (event.key === "Escape" && !infoOpen) selectPlace(null);
  });
  useEffect(() => {
    if (!selectedPlace) return;
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [selectedPlace]);

  const allActive = activeSlugs.length === 0;
  const areaName = subDistrict?.name ?? "Jakarta";

  let emptyState: EmptyState | null = null;
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
            onAction: () => updateParams({ kategori: undefined }),
          };
  }

  return (
    <div className="flex min-h-dvh flex-col md:h-dvh">
      <HomeHeader
        subDistricts={subDistricts}
        subDistrict={subDistrict}
        draft={draft}
        onDraftChange={setDraft}
        onSearch={() => updateParams({ q: draft.trim() || undefined })}
        onClearSearch={clearSearch}
        onKecamatanChange={(slug) =>
          updateParams({ kecamatan: slug, tempat: undefined, limit: undefined })
        }
        onOpenInfo={() => setInfoOpen(true)}
      />

      <CategoryNav
        categories={categories}
        activeSlugs={activeSlugs}
        onToggle={toggleCategory}
        onShowAll={() => updateParams({ kategori: undefined })}
        placeCount={visiblePlaces.length}
        areaName={areaName}
        query={query}
      />

      {/* Map + menu board */}
      <main className="flex flex-1 flex-col md:grid md:min-h-0 md:grid-cols-[2fr_1fr]">
        <section
          aria-label="Peta tempat penting"
          className="md:border-ink relative h-[58dvh] min-h-80 md:h-auto md:border-r-2"
        >
          <SurvivalMap
            places={visiblePlaces}
            focusKey={subDistrict?.slug ?? ""}
            selectedId={selectedPlace?.id ?? null}
            onSelect={selectPlace}
            onLimitChange={setLimit}
            sheetRef={sheetRef}
          >
            {emptyState ? <MapEmptyState {...emptyState} /> : null}
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
                fields={selectedFields}
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

function HomeError() {
  return <ErrorBoard title="Peta belum bisa dimuat." href="/" />;
}
