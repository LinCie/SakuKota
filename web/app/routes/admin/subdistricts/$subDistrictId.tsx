import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { SubDistrictDetailPage } from "~/features/subdistricts/pages/subdistrict-detail-page";
import {
  subDistrictQuery,
  useSubDistrict,
} from "~/features/subdistricts/queries";
import { LoadingState } from "~/shared/components/loading-state";

export const Route = createFileRoute("/admin/subdistricts/$subDistrictId")({
  loader: ({ context: { queryClient }, params }) =>
    queryClient.ensureQueryData(subDistrictQuery(Number(params.subDistrictId))),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  component: SubDistrictDetailRoute,
});

function SubDistrictDetailRoute() {
  const { subDistrictId } = Route.useParams();
  const { data: subDistrict } = useSubDistrict(Number(subDistrictId));
  return <SubDistrictDetailPage subDistrict={subDistrict} />;
}
