import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { AdminDetailSkeleton } from "~/features/admin/components/admin-skeletons";
import { SubDistrictDetailPage } from "~/features/subdistricts/pages/subdistrict-detail-page";
import {
  subDistrictQuery,
  subDistrictsQuery,
  useSubDistrict,
  useSubDistricts,
} from "~/features/subdistricts/queries";

export const Route = createFileRoute("/admin/subdistricts/$subDistrictId")({
  loader: ({ context: { queryClient }, params }) =>
    Promise.all([
      queryClient.query({
        ...subDistrictQuery(Number(params.subDistrictId)),
        staleTime: "static",
      }),
      queryClient.query({ ...subDistrictsQuery(), staleTime: "static" }),
    ]),
  pendingComponent: () => (
    <AdminDetailSkeleton
      actions={2}
      cards={[
        { title: "Informasi", content: 5 },
        { title: "Batas wilayah", content: "image" },
        { title: "Foto kecamatan", content: "image" },
      ]}
    />
  ),
  errorComponent: AdminRouteError,
  component: SubDistrictDetailRoute,
});

function SubDistrictDetailRoute() {
  const { subDistrictId } = Route.useParams();
  const { data: subDistrict } = useSubDistrict(Number(subDistrictId));
  const { data: subDistricts } = useSubDistricts();
  return (
    <SubDistrictDetailPage
      subDistrict={subDistrict}
      subDistricts={subDistricts}
    />
  );
}
