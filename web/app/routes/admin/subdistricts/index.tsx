import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { AdminListSkeleton } from "~/features/admin/components/admin-skeletons";
import { SubDistrictsPage } from "~/features/subdistricts/pages/subdistricts-page";
import {
  subDistrictsQuery,
  useSubDistricts,
} from "~/features/subdistricts/queries";

export const Route = createFileRoute("/admin/subdistricts/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.query({ ...subDistrictsQuery(), staleTime: "static" }),
  pendingComponent: () => <AdminListSkeleton action />,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Kecamatan · Admin SakuKota" }] }),
  component: SubDistrictsRoute,
});

function SubDistrictsRoute() {
  const { data: subDistricts } = useSubDistricts();
  return <SubDistrictsPage subDistricts={subDistricts} />;
}
