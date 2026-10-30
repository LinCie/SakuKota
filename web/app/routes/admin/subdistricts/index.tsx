import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { SubDistrictsPage } from "~/features/subdistricts/pages/subdistricts-page";
import {
  subDistrictsQuery,
  useSubDistricts,
} from "~/features/subdistricts/queries";
import { LoadingState } from "~/shared/components/loading-state";

export const Route = createFileRoute("/admin/subdistricts/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(subDistrictsQuery()),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Kecamatan · Admin Rantau" }] }),
  component: SubDistrictsRoute,
});

function SubDistrictsRoute() {
  const { data: subDistricts } = useSubDistricts();
  return <SubDistrictsPage subDistricts={subDistricts} />;
}
