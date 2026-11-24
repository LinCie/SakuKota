import { createFileRoute } from "@tanstack/react-router";
import { useEffect } from "react";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { AdminListSkeleton } from "~/features/admin/components/admin-skeletons";
import { PlacesPage } from "~/features/places/pages/places-page";
import {
  categoriesQuery,
  placesQuery,
  useCategories,
  usePlaces,
} from "~/features/places/queries";
import {
  subDistrictsQuery,
  useSubDistricts,
} from "~/features/subdistricts/queries";

const PAGE_SIZE = 25;

type PlacesSearch = {
  /** The page of the table, counting from 1; absent is the first page. */
  page?: number;
  /** Searches the places; absent lists them all. */
  q?: string;
};

// Search params arrive JSON-parsed (`?q=2024` is a number). Malformed values
// are dropped instead of erroring.
function placesParams({ page, q }: PlacesSearch) {
  return {
    q,
    limit: PAGE_SIZE,
    offset: ((page ?? 1) - 1) * PAGE_SIZE,
  };
}

export const Route = createFileRoute("/admin/places/")({
  validateSearch: (search: Record<string, unknown>): PlacesSearch => {
    const { page, q } = search;
    const text =
      typeof q === "string" || typeof q === "number" ? String(q).trim() : "";
    return {
      page:
        typeof page === "number" && Number.isInteger(page) && page > 1
          ? page
          : undefined,
      q: text || undefined,
    };
  },
  loaderDeps: ({ search: { page, q } }) => ({ page, q }),
  loader: ({ context: { queryClient }, deps }) =>
    Promise.all([
      queryClient.query({ ...categoriesQuery(), staleTime: "static" }),
      queryClient.query({ ...subDistrictsQuery(), staleTime: "static" }),
      queryClient.query({
        ...placesQuery(placesParams(deps)),
        staleTime: "static",
      }),
    ]),
  pendingComponent: () => <AdminListSkeleton action />,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Tempat · Admin SakuKota" }] }),
  component: PlacesRoute,
});

function PlacesRoute() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const { data: categories } = useCategories();
  const { data: subDistricts } = useSubDistricts();
  const {
    data: { places, total },
  } = usePlaces(placesParams(search));

  const page = search.page ?? 1;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));

  // Deleting the last place of the last page leaves that page empty.
  useEffect(() => {
    if (page > pageCount) {
      void navigate({
        search: (previous) => ({
          ...previous,
          page: pageCount > 1 ? pageCount : undefined,
        }),
        replace: true,
      });
    }
  }, [page, pageCount, navigate]);

  return (
    <PlacesPage
      places={places}
      total={total}
      subDistricts={subDistricts}
      categories={categories}
      query={search.q ?? ""}
      page={page}
      pageCount={pageCount}
      onQueryChange={(q) =>
        void navigate({
          search: { q: q || undefined },
          replace: true,
          resetScroll: false,
        })
      }
      onPageChange={(next) =>
        void navigate({
          search: (previous) => ({
            ...previous,
            page: next > 1 ? next : undefined,
          }),
        })
      }
    />
  );
}
