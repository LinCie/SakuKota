import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { getSubDistrict } from "~/features/subdistricts/api";
import { SubDistrictDetailPage } from "~/features/subdistricts/pages/subdistrict-detail-page";
import { LoadingState } from "~/shared/components/loading-state";
import { unwrap } from "~/shared/api";

export const Route = createFileRoute("/admin/subdistricts/$subDistrictId")({
  loader: async ({ params }) => ({
    subDistrict: unwrap(await getSubDistrict(Number(params.subDistrictId))),
  }),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  component: SubDistrictDetailRoute,
});

function SubDistrictDetailRoute() {
  return <SubDistrictDetailPage {...Route.useLoaderData()} />;
}
