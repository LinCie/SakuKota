import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { categoriesQuery, useCategories } from "~/features/places/queries";
import {
  subDistrictsQuery,
  useSubDistricts,
} from "~/features/subdistricts/queries";
import { SuggestionsPage } from "~/features/suggestions/pages/suggestions-page";
import {
  suggestionsQuery,
  useSuggestions,
} from "~/features/suggestions/queries";
import {
  SUGGESTION_STATUSES,
  type SuggestionStatus,
} from "~/features/suggestions/types";
import { LoadingState } from "~/shared/components/loading-state";

type SuggestionsSearch = {
  /** Filters to one status; absent shows every status. */
  status?: SuggestionStatus;
};

export const Route = createFileRoute("/admin/suggestions/")({
  // An unknown status is dropped instead of erroring.
  validateSearch: (search: Record<string, unknown>): SuggestionsSearch => ({
    status: SUGGESTION_STATUSES.find((status) => status === search.status),
  }),
  loaderDeps: ({ search: { status } }) => ({ status }),
  loader: ({ context: { queryClient }, deps }) =>
    Promise.all([
      queryClient.query({ ...suggestionsQuery(deps), staleTime: "static" }),
      queryClient.query({ ...categoriesQuery(), staleTime: "static" }),
      queryClient.query({ ...subDistrictsQuery(), staleTime: "static" }),
    ]),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Usulan · Admin Rantau" }] }),
  component: SuggestionsRoute,
});

function SuggestionsRoute() {
  const { status } = Route.useSearch();
  const navigate = Route.useNavigate();
  const { data: suggestions } = useSuggestions({ status });
  const { data: categories } = useCategories();
  const { data: subDistricts } = useSubDistricts();

  return (
    <SuggestionsPage
      suggestions={suggestions}
      categories={categories}
      subDistricts={subDistricts}
      status={status}
      onStatusChange={(next) =>
        void navigate({ search: { status: next }, replace: true })
      }
    />
  );
}
