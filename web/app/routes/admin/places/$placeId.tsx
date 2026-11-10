import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { AdminDetailSkeleton } from "~/features/admin/components/admin-skeletons";
import { PlaceDetailPage } from "~/features/places/pages/place-detail-page";
import {
  ensurePlaceData,
  usePlaceData,
} from "~/features/places/load-place-data";
import { placeQuery, usePlace } from "~/features/places/queries";

export const Route = createFileRoute("/admin/places/$placeId")({
  loader: ({ context: { queryClient }, params }) =>
    Promise.all([
      queryClient.query({
        ...placeQuery(Number(params.placeId)),
        staleTime: "static",
      }),
      ensurePlaceData(queryClient),
    ]),
  pendingComponent: () => (
    <AdminDetailSkeleton
      actions={2}
      cards={[
        { title: "Informasi", content: 7 },
        { title: "Khusus kategori", content: 4 },
        { title: "Foto tempat", content: "image" },
      ]}
    />
  ),
  errorComponent: AdminRouteError,
  component: PlaceDetailRoute,
});

function PlaceDetailRoute() {
  const { placeId } = Route.useParams();
  const { data: place } = usePlace(Number(placeId));
  return <PlaceDetailPage place={place} {...usePlaceData()} />;
}
