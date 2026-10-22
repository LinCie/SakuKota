import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { getSubDistricts } from "~/features/subdistricts/api";
import { SubDistrictsPage } from "~/features/subdistricts/components/subdistricts-page";
import { LoadingState } from "~/shared/components/loading-state";
import { unwrap } from "~/shared/api";

export const Route = createFileRoute("/admin/subdistricts/")({
  loader: async () => ({ subDistricts: unwrap(await getSubDistricts()) }),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Kecamatan · Admin Rantau" }] }),
  component: SubDistrictsRoute,
});

function SubDistrictsRoute() {
  return <SubDistrictsPage {...Route.useLoaderData()} />;
}
