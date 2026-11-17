import { createFileRoute } from "@tanstack/react-router";

import {
  categoriesQuery,
  placesInfiniteQuery,
  placesQuery,
} from "~/features/places/queries";
import {
  HomeError,
  HomePage,
  placesSelection,
  type HomeSearch,
} from "~/features/places/pages/home-page";
import { subDistrictsQuery } from "~/features/subdistricts/queries";

// Search params arrive JSON-parsed (`?q=2024` is a number). Malformed values
// are dropped instead of erroring.
function searchString(value: unknown) {
  if (typeof value === "number") return String(value);
  return typeof value === "string" && value !== "" ? value : undefined;
}

const BBOX_COORDINATE = /^-?\d+(\.\d+)?$/;

// A bbox must be "west,south,east,north" with west <= east and south <= north,
// within valid coordinates; anything else is dropped.
function searchBBox(value: unknown) {
  if (typeof value !== "string") return undefined;
  const parts = value.split(",");
  if (parts.length !== 4 || !parts.every((part) => BBOX_COORDINATE.test(part)))
    return undefined;
  const [west, south, east, north] = parts.map(Number);
  return -180 <= west &&
    west <= east &&
    east <= 180 &&
    -90 <= south &&
    south <= north &&
    north <= 90
    ? value
    : undefined;
}

// A limit must be a whole number from 1 to 100, the most the server returns.
function searchLimit(value: unknown) {
  return typeof value === "number" &&
    Number.isInteger(value) &&
    1 <= value &&
    value <= 100
    ? value
    : undefined;
}

export const Route = createFileRoute("/")({
  validateSearch: (search: Record<string, unknown>): HomeSearch => ({
    kecamatan: searchString(search.kecamatan),
    kategori: searchString(search.kategori),
    q: searchString(search.q),
    bbox: searchBBox(search.bbox),
    limit: searchLimit(search.limit),
    tempat: typeof search.tempat === "number" ? search.tempat : undefined,
  }),
  // The kecamatan, categories, and search are applied by the backend; the
  // selected place (`tempat`) is picked from the loaded results, so changing
  // it doesn't reload.
  loaderDeps: ({ search: { kecamatan, kategori, q, bbox, limit } }) => ({
    kecamatan,
    kategori,
    q,
    bbox,
    limit,
  }),
  loader: async ({ context: { queryClient }, deps }) => {
    // The map needs Leaflet; fetch it alongside the data instead of after.
    void import("leaflet");
    const [categories, subDistricts] = await Promise.all([
      queryClient.query({ ...categoriesQuery(), staleTime: "static" }),
      queryClient.query({ ...subDistrictsQuery(), staleTime: "static" }),
    ]);
    const { params, listParams } = placesSelection(
      deps,
      categories,
      subDistricts,
    );
    await Promise.all([
      queryClient.query({ ...placesQuery(params), staleTime: "static" }),
      queryClient.infiniteQuery({
        ...placesInfiniteQuery(listParams),
        staleTime: "static",
      }),
    ]);
  },
  head: () => ({
    meta: [
      { title: "Rantau — Peta bertahan hidup perantau Jakarta" },
      {
        name: "description",
        content:
          "Warung, penjahit, klinik, laundry, apotek, masjid, dan kebutuhan harian perantau lainnya di sekitarmu, plus info banjir, keamanan, dan biaya harian per kecamatan.",
      },
    ],
  }),
  component: HomeRoute,
  errorComponent: HomeError,
});

function HomeRoute() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  return (
    <HomePage
      search={search}
      onSearchChange={(changes) =>
        void navigate({
          search: (previous) => ({ ...previous, ...changes }),
          replace: true,
          resetScroll: false,
        })
      }
    />
  );
}
