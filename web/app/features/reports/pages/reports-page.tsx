import { useNavigate } from "@tanstack/react-router";

import { SelectControl } from "~/shared/components/field";
import { PageHeader } from "~/shared/components/page-header";
import { formatNumber } from "~/shared/format";

import { ReportsTable, STATUS_LABELS } from "../components/reports-table";
import { REPORT_STATUSES, type Report, type ReportStatus } from "../types";

type ReportsPageProps = {
  /** Oldest first. */
  reports: Report[];
  /** The status the list is filtered to; absent shows every status. */
  status: ReportStatus | undefined;
  onStatusChange: (status: ReportStatus | undefined) => void;
};

/** The reports list: a status filter and a table to open one with its content. */
export function ReportsPage({
  reports,
  status,
  onStatusChange,
}: ReportsPageProps) {
  const navigate = useNavigate();

  return (
    <>
      <PageHeader
        title="Laporan"
        description="Tempat dan ulasan yang dilaporkan pengguna sebagai penyalahgunaan."
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="w-full sm:w-44">
          <SelectControl
            aria-label="Status"
            value={status ?? ""}
            onValueChange={(value) =>
              onStatusChange(
                REPORT_STATUSES.find((s) => s === value) ?? undefined,
              )
            }
            placeholder="Semua status"
            options={REPORT_STATUSES.map((s) => ({
              value: s,
              label: STATUS_LABELS[s],
            }))}
            size="sm"
          />
        </div>
        <p className="text-muted-foreground text-sm tabular-nums">
          {formatNumber(reports.length)} laporan
        </p>
      </div>

      <ReportsTable
        reports={reports}
        onView={(report) =>
          void navigate({
            to: "/admin/reports/$reportId",
            params: { reportId: String(report.id) },
          })
        }
      />
    </>
  );
}
