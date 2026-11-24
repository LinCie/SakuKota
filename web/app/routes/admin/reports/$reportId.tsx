import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { AdminDetailSkeleton } from "~/features/admin/components/admin-skeletons";
import { ReportDetailPage } from "~/features/reports/pages/report-detail-page";
import { reportQuery, useReport } from "~/features/reports/queries";

export const Route = createFileRoute("/admin/reports/$reportId")({
  loader: ({ context: { queryClient }, params }) =>
    queryClient.query({
      ...reportQuery(Number(params.reportId)),
      staleTime: "static",
    }),
  pendingComponent: () => (
    <AdminDetailSkeleton
      actions={2}
      cards={[
        { title: "Konten", content: 4 },
        { title: "Laporan", content: 5 },
      ]}
    />
  ),
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Laporan · Admin SakuKota" }] }),
  component: ReportDetailRoute,
});

function ReportDetailRoute() {
  const { reportId } = Route.useParams();
  const { data: report } = useReport(Number(reportId));
  return <ReportDetailPage report={report} />;
}
