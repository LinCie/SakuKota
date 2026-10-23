import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { getPlace } from "~/features/places/api";
import { PlaceDetailPage } from "~/features/places/pages/place-detail-page";
import { loadPlaceData } from "~/features/places/load-place-data";
import { LoadingState } from "~/shared/components/loading-state";
import { unwrap } from "~/shared/api";

export const Route = createFileRoute("/admin/places/$placeId")({
  loader: async ({ params }) => {
    const [place, data] = await Promise.all([
      getPlace(Number(params.placeId)),
      loadPlaceData(),
    ]);
    return { place: unwrap(place), ...data };
  },
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  component: PlaceDetailRoute,
});

function PlaceDetailRoute() {
  return <PlaceDetailPage {...Route.useLoaderData()} />;
}
