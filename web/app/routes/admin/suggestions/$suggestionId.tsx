import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import {
  ensurePlaceData,
  usePlaceData,
} from "~/features/places/load-place-data";
import { SuggestionDetailPage } from "~/features/suggestions/pages/suggestion-detail-page";
import { suggestionQuery, useSuggestion } from "~/features/suggestions/queries";
import { LoadingState } from "~/shared/components/loading-state";

export const Route = createFileRoute("/admin/suggestions/$suggestionId")({
  // The places are for the approve form's map.
  loader: ({ context: { queryClient }, params }) =>
    Promise.all([
      queryClient.query({
        ...suggestionQuery(Number(params.suggestionId)),
        staleTime: "static",
      }),
      ensurePlaceData(queryClient),
    ]),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Usulan · Admin Rantau" }] }),
  component: SuggestionDetailRoute,
});

function SuggestionDetailRoute() {
  const { suggestionId } = Route.useParams();
  const { data: suggestion } = useSuggestion(Number(suggestionId));
  return <SuggestionDetailPage suggestion={suggestion} {...usePlaceData()} />;
}
