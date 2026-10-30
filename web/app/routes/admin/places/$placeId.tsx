import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { PlaceDetailPage } from "~/features/places/pages/place-detail-page";
import {
  ensurePlaceData,
  usePlaceData,
} from "~/features/places/load-place-data";
import { placeQuery, usePlace } from "~/features/places/queries";
import { LoadingState } from "~/shared/components/loading-state";

export const Route = createFileRoute("/admin/places/$placeId")({
  loader: ({ context: { queryClient }, params }) =>
    Promise.all([
      queryClient.ensureQueryData(placeQuery(Number(params.placeId))),
      ensurePlaceData(queryClient),
    ]),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  component: PlaceDetailRoute,
});

function PlaceDetailRoute() {
  const { placeId } = Route.useParams();
  const { data: place } = usePlace(Number(placeId));
  return <PlaceDetailPage place={place} {...usePlaceData()} />;
}
