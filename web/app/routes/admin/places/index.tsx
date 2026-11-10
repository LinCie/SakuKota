import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { AdminListSkeleton } from "~/features/admin/components/admin-skeletons";
import { PlacesPage } from "~/features/places/pages/places-page";
import {
  ensurePlaceData,
  usePlaceData,
} from "~/features/places/load-place-data";

export const Route = createFileRoute("/admin/places/")({
  loader: ({ context: { queryClient } }) => ensurePlaceData(queryClient),
  pendingComponent: () => <AdminListSkeleton action />,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Tempat · Admin Rantau" }] }),
  component: PlacesRoute,
});

function PlacesRoute() {
  return <PlacesPage {...usePlaceData()} />;
}
