import { useQuery } from "@tanstack/react-query";
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
import { ErrorBoard } from "~/shared/components/error-board";

export type HomeSearch = {
  kecamatan?: string;
  /** Comma-separated category slugs; absent means every category. */
  kategori?: string;
  q?: string;
  /** Caps the places to the lowest ids while the map is zoomed out. */
  limit?: number;
  /** The map's view as "west,south,east,north"; only places inside it are loaded. */
  bbox?: string;
  tempat?: number;
};

function activeCategorySlugs(kategori: string | undefined) {
  return (kategori ?? "").split(",").filter(Boolean);
}

type PlacesFilters = Pick<
  HomeSearch,
  "kecamatan" | "kategori" | "q" | "limit" | "bbox"
>;

/**
 * The kecamatan on show (the one in the URL, else the first) and the places
 * queries for the filters: `params` for the map, which follows its view, and
 * `listParams` for the list, which does not. With no kecamatan at all there are
 * no places either.
 */
export function placesSelection(
  { kecamatan, kategori, q, limit, bbox }: PlacesFilters,
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
  const params: GetPlacesParams = { ...listParams, limit, bbox };
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

  function setView({ bbox, limit }: MapView) {
    if (bbox !== search.bbox || limit !== search.limit) {
      updateParams({ bbox, limit });
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
          // The new kecamatan loads whole, so the map can frame it.
          updateParams({
            kecamatan: slug,
            tempat: undefined,
            limit: undefined,
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
      />

      {/* Map + menu board */}
      <main className="flex flex-1 flex-col md:grid md:min-h-0 md:grid-cols-[2fr_1fr]">
        <section
          aria-label="Peta tempat penting"
          className="md:border-ink relative h-[58dvh] min-h-80 md:h-auto md:border-r-2"
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
            <div
              ref={sheetRef}
              className="border-ink bg-paint fixed inset-x-0 bottom-0 z-1100 max-h-[60dvh] overflow-y-auto overscroll-contain rounded-t-[10px] border-2 border-b-0 shadow-[0_-10px_30px_rgb(26_26_23/0.3)] motion-safe:transition-transform motion-safe:duration-300 motion-safe:ease-out md:static md:z-auto md:max-h-none md:overflow-visible md:rounded-none md:border-0 md:shadow-none motion-safe:starting:translate-y-full md:motion-safe:starting:translate-y-0"
            >
              <PlaceDetails
                place={selectedPlace}
                fields={selectedFields}
                onClose={() => selectPlace(null)}
              />
              <PlaceReviews key={selectedPlace.id} placeId={selectedPlace.id} />
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

export function HomeError() {
  return <ErrorBoard title="Peta belum bisa dimuat." href="/" />;
}
