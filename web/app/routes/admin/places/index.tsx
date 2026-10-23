import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { PlacesPage } from "~/features/places/pages/places-page";
import { loadPlaceData } from "~/features/places/load-place-data";
import { LoadingState } from "~/shared/components/loading-state";

export const Route = createFileRoute("/admin/places/")({
  loader: loadPlaceData,
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Tempat · Admin Rantau" }] }),
  component: PlacesRoute,
});

function PlacesRoute() {
  return <PlacesPage {...Route.useLoaderData()} />;
}
