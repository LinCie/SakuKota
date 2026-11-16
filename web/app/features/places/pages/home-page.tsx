import { useQuery } from "@tanstack/react-query";
import { Info, List, X } from "lucide-react";
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
import { SearchField } from "~/features/places/components/search-field";
import {
  SurvivalMap,
  type FlyRequest,
  type MapView,
} from "~/features/places/components/survival-map";
import {
  placeQuery,
  useCategories,
  usePlaces,
  usePlacesInfinite,
  type PlaceListParams,
} from "~/features/places/queries";
import type { CategoryDetail, Place } from "~/features/places/types";
import { PlaceReviews } from "~/features/reviews/components/place-reviews";
import { useSubDistricts } from "~/features/subdistricts/queries";
import type { SubDistrict } from "~/features/subdistricts/types";
import { KecamatanBoard } from "~/features/subdistricts/components/kecamatan-board";
import { cn } from "~/shared/cn";
import { ErrorBoard } from "~/shared/components/error-board";
import {
  menuButtonClass,
  signButtonClass,
} from "~/shared/components/menu-board";

export type HomeSearch = {
  kecamatan?: string;
  /** Comma-separated category slugs; absent means every category. */
  kategori?: string;
  q?: string;
  /** The map's view as "west,south,east,north"; only places inside it are loaded. */
  bbox?: string;
  tempat?: number;
};

function activeCategorySlugs(kategori: string | undefined) {
  return (kategori ?? "").split(",").filter(Boolean);
}

type PlacesFilters = Pick<HomeSearch, "kecamatan" | "kategori" | "q" | "bbox">;

/**
 * The kecamatan on show (the one in the URL, else the first) and the places
 * queries for the filters: `params` for the map, which follows its view, and
 * `listParams` for the list, which does not. With no kecamatan at all there are
 * no places either.
 */
export function placesSelection(
  { kecamatan, kategori, q, bbox }: PlacesFilters,
  categories: CategoryDetail[],
  subDistricts: SubDistrict[],
) {
  const subDistrict =
    subDistricts.find((sd) => sd.slug === kecamatan) ?? subDistricts[0];
  const activeSlugs = activeCategorySlugs(kategori);
  const listParams: PlaceListParams = {
    subDistrictId: subDistrict?.id,
    categoryIds: categories
      .filter((category) => activeSlugs.includes(category.slug))
      .map((category) => category.id),
    q: q?.trim(),
  };
  const params: GetPlacesParams = { ...listParams, bbox };
  return { subDistrict, params, listParams };
}

type HomePageProps = {
  search: HomeSearch;
  /** Merges the changes into the URL's search params. */
  onSearchChange: (changes: HomeSearch) => void;
};

export function HomePage({ search, onSearchChange }: HomePageProps) {
  const { data: categories } = useCategories();
  const { data: subDistricts } = useSubDistricts();
  const { subDistrict, params, listParams } = placesSelection(
    search,
    categories,
    subDistricts,
  );
  // The map's places follow its view; the list's do not.
  const {
    data: { places: mapPlaces },
  } = usePlaces(params);
  const {
    data: list,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isFetchNextPageError,
  } = usePlacesInfinite(listParams);
  const listPlaces = list.pages.flatMap((page) => page.places);
  const listTotal = list.pages[0]?.total ?? 0;
  const [flyRequest, setFlyRequest] = useState<FlyRequest | null>(null);
  const [infoOpen, setInfoOpen] = useState(false);
  // The list is a side column from 768px, where it starts open; on phones it is a bottom sheet that starts closed.
  const [listOpen, setListOpen] = useState(
    () => window.matchMedia("(min-width: 768px)").matches,
  );
  const sheetRef = useRef<HTMLDivElement>(null);

  const activeSlugs = activeCategorySlugs(search.kategori);
  const query = search.q ?? "";

  const [draft, setDraft] = useState(query);

  function updateParams(changes: HomeSearch) {
    onSearchChange(changes);
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
    listPlaces.find((place) => place.id === search.tempat) ??
    mapPlaces.find((place) => place.id === search.tempat) ??
    null;
  // The selected place may be on neither (a link, or a filter that leaves it out, or a page not loaded yet); it is fetched so its details stay open.
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

  // Picking from the list teleports the map to the place.
  function selectListedPlace(place: Place) {
    setFlyRequest({ place });
    selectPlace(place.id);
  }

  function setView({ bbox }: MapView) {
    if (bbox !== search.bbox) {
      updateParams({ bbox });
    }
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

  const panelOpen = listOpen || selectedPlace !== null;

  function togglePanel() {
    if (panelOpen) {
      setListOpen(false);
      selectPlace(null);
    } else {
      setListOpen(true);
    }
  }

  const allActive = activeSlugs.length === 0;
  const areaName = subDistrict?.name ?? "Jakarta";

  let emptyState: EmptyState | null = null;
  if (listTotal === 0) {
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

  // Rendered twice: in the header from 640px, in the category strip below.
  const searchField = (
    <SearchField
      draft={draft}
      onDraftChange={setDraft}
      onSearch={() => updateParams({ q: draft.trim() || undefined })}
      onClearSearch={clearSearch}
    />
  );

  return (
    <div className="flex h-dvh flex-col overflow-hidden">
      <HomeHeader
        subDistricts={subDistricts}
        subDistrict={subDistrict}
        search={searchField}
        onKecamatanChange={(slug) =>
          // The new kecamatan loads without a view, so the map can frame it.
          updateParams({
            kecamatan: slug,
            tempat: undefined,
            bbox: undefined,
          })
        }
        onOpenInfo={() => setInfoOpen(true)}
      />

      <CategoryNav
        categories={categories}
        activeSlugs={activeSlugs}
        onToggle={toggleCategory}
        onShowAll={() => updateParams({ kategori: undefined })}
        count={listTotal}
        areaName={areaName}
        query={query}
        search={searchField}
      />

      {/* The map takes the rest of the screen; the list and details sit beside it or over it. */}
      <main className="flex min-h-0 flex-1">
        <section
          aria-label="Peta tempat penting"
          className="relative min-w-0 flex-1"
        >
          <SurvivalMap
            places={mapPlaces}
            focusKey={subDistrict?.slug ?? ""}
            selectedId={selectedPlace?.id ?? null}
            onSelect={selectPlace}
            flyRequest={flyRequest}
            onViewChange={setView}
            sheetRef={sheetRef}
          >
            {/* On phones "Lihat info" leaves the header for the map. */}
            {subDistrict ? (
              <button
                type="button"
                onClick={() => setInfoOpen(true)}
                className={cn(
                  signButtonClass,
                  "absolute top-3 right-3 z-1000 inline-flex h-10 items-center gap-2 px-3 shadow-[0_3px_8px_rgb(26_26_23/0.25)] sm:hidden",
                )}
              >
                <Info aria-hidden="true" className="size-4" strokeWidth={2.5} />
                Info kecamatan
              </button>
            ) : null}
            <button
              type="button"
              onClick={togglePanel}
              aria-expanded={panelOpen}
              className={cn(
                "border-ink bg-paint font-sign text-ink hover:bg-sign-soft focus-visible:outline-cat absolute bottom-6 left-3 z-1000 inline-flex h-11 items-center gap-2 rounded-[3px] border-2 px-4 text-sm shadow-[0_3px_8px_rgb(26_26_23/0.25)] transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 md:top-3 md:right-3 md:bottom-auto md:left-auto md:h-10",
                panelOpen && "max-md:hidden",
              )}
            >
              <List aria-hidden="true" className="size-4" strokeWidth={2.5} />
              {panelOpen ? "Sembunyikan daftar" : "Daftar tempat"}
              {panelOpen ? null : (
                <span className="tabular-nums md:hidden">{listTotal}</span>
              )}
            </button>
            {emptyState ? <MapEmptyState {...emptyState} /> : null}
          </SurvivalMap>
        </section>

        {panelOpen ? (
          <aside
            ref={sheetRef}
            aria-label="Daftar dan detail tempat"
            className="border-ink bg-paint fixed inset-x-0 bottom-0 z-1100 max-h-[60dvh] overflow-y-auto overscroll-contain rounded-t-[10px] border-2 border-b-0 shadow-[0_-10px_30px_rgb(26_26_23/0.3)] motion-safe:transition-transform motion-safe:duration-300 motion-safe:ease-out md:static md:z-auto md:max-h-none md:w-96 md:shrink-0 md:overflow-y-auto md:rounded-none md:border-0 md:border-l-2 md:shadow-none motion-safe:starting:translate-y-full md:motion-safe:starting:translate-y-0"
          >
            <div className={selectedPlace ? "hidden" : undefined}>
              <div className="border-ink bg-paint sticky top-0 z-10 flex items-center justify-between gap-3 border-b-2 py-2 pr-2 pl-4 md:px-5 md:py-3">
                <h2 className="font-sign text-ink text-base">Daftar tempat</h2>
                <div className="flex items-center gap-2">
                  {/* The strip's count is for screen readers only on phones. */}
                  <p
                    aria-hidden="true"
                    className="text-ink-soft text-sm sm:hidden"
                  >
                    <span className="font-sign text-ink text-base tabular-nums">
                      {listTotal}
                    </span>{" "}
                    tempat
                  </p>
                  <button
                    type="button"
                    onClick={() => setListOpen(false)}
                    aria-label="Tutup daftar tempat"
                    className={cn(
                      menuButtonClass,
                      "grid size-9 place-items-center rounded-[3px] md:hidden",
                    )}
                  >
                    <X
                      aria-hidden="true"
                      className="size-5"
                      strokeWidth={2.5}
                    />
                  </button>
                </div>
              </div>
              <PlaceList
                places={listPlaces}
                onSelect={selectListedPlace}
                emptyMessage={emptyState?.message ?? ""}
                hasMore={hasNextPage}
                loadingMore={isFetchingNextPage}
                loadMoreFailed={isFetchNextPageError}
                onLoadMore={() => void fetchNextPage()}
              />
            </div>

            {selectedPlace ? (
              <>
                <PlaceDetails
                  place={selectedPlace}
                  fields={selectedFields}
                  onClose={() => selectPlace(null)}
                />
                <PlaceReviews
                  key={selectedPlace.id}
                  placeId={selectedPlace.id}
                />
              </>
            ) : null}
          </aside>
        ) : null}
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

export function HomeError() {
  return <ErrorBoard title="Peta belum bisa dimuat." href="/" />;
}
