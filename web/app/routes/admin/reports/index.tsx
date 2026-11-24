import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { AdminListSkeleton } from "~/features/admin/components/admin-skeletons";
import { ReportsPage } from "~/features/reports/pages/reports-page";
import { reportsQuery, useReports } from "~/features/reports/queries";
import { REPORT_STATUSES, type ReportStatus } from "~/features/reports/types";

type ReportsSearch = {
  /** Filters to one status; absent shows every status. */
  status?: ReportStatus;
};

export const Route = createFileRoute("/admin/reports/")({
  // An unknown status is dropped instead of erroring.
  validateSearch: (search: Record<string, unknown>): ReportsSearch => ({
    status: REPORT_STATUSES.find((status) => status === search.status),
  }),
  loaderDeps: ({ search: { status } }) => ({ status }),
  loader: ({ context: { queryClient }, deps }) =>
    queryClient.query({ ...reportsQuery(deps), staleTime: "static" }),
  pendingComponent: () => <AdminListSkeleton filter />,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Laporan · Admin Rantau" }] }),
  component: ReportsRoute,
});

function ReportsRoute() {
  const { status } = Route.useSearch();
  const navigate = Route.useNavigate();
  const { data: reports } = useReports({ status });

  return (
    <ReportsPage
      reports={reports}
      status={status}
      onStatusChange={(next) =>
        void navigate({ search: { status: next }, replace: true })
      }
    />
  );
}
