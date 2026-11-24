import { Eye } from "lucide-react";

import { Badge } from "~/shared/components/ui/badge";
import { DataTable, type Column } from "~/shared/components/data-table";
import { RowActionsMenu } from "~/shared/components/row-actions-menu";
import { formatLocalDate } from "~/shared/format";

import { REASON_LABELS, type Report, type ReportStatus } from "../types";

export const STATUS_LABELS: Record<ReportStatus, string> = {
  pending: "Menunggu",
  removed: "Konten dihapus",
  dismissed: "Diabaikan",
};

// As for suggestions: a pending report is still open, so it stays quiet; a
// resolved one takes its outcome's tint.
const STATUS_CLASSES: Record<ReportStatus, string> = {
  pending: "bg-muted text-muted-foreground",
  removed: "bg-sambal-soft text-sambal-deep",
  dismissed: "bg-pandan-soft text-pandan-deep",
};

export function ReportStatusBadge({ status }: { status: ReportStatus }) {
  return (
    <Badge className={STATUS_CLASSES[status]}>{STATUS_LABELS[status]}</Badge>
  );
}

/** What the report is about, e.g. "Ulasan #9 di tempat #4". */
export function reportTarget(report: Report) {
  return report.review_id === null
    ? `Tempat #${report.place_id}`
    : `Ulasan #${report.review_id} di tempat #${report.place_id}`;
}

type ReportsTableProps = {
  reports: Report[];
  onView: (report: Report) => void;
};

/** Reports, with a row action to open one with its content. */
export function ReportsTable({ reports, onView }: ReportsTableProps) {
  const columns: Column<Report>[] = [
    {
      key: "target",
      header: "Konten",
      pinned: true,
      className: "min-w-48",
      cell: (report) => (
        <span className="font-semibold">{reportTarget(report)}</span>
      ),
    },
    {
      key: "reason",
      header: "Alasan",
      className: "min-w-56",
      cell: (report) => (
        <div className="flex min-w-0 flex-col gap-0.5">
          <span>{REASON_LABELS[report.reason]}</span>
          {report.note ? (
            <span
              title={report.note}
              className="text-muted-foreground max-w-80 truncate text-xs"
            >
              {report.note}
            </span>
          ) : null}
        </div>
      ),
    },
    {
      key: "author",
      header: "Pelapor",
      cell: (report) => report.user.name,
    },
    {
      key: "created",
      header: "Dikirim",
      cell: (report) => (
        <time dateTime={report.created_at} className="tabular-nums">
          {formatLocalDate(new Date(report.created_at))}
        </time>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (report) => <ReportStatusBadge status={report.status} />,
    },
    {
      key: "actions",
      header: "Aksi",
      className: "w-12 text-right",
      cell: (report) => (
        <RowActionsMenu
          label={`Aksi untuk laporan ${reportTarget(report)}`}
          actions={[
            { label: "Lihat", icon: Eye, onSelect: () => onView(report) },
          ]}
        />
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      rows={reports}
      getRowKey={(report) => report.id}
      empty="Belum ada laporan yang cocok."
    />
  );
}
