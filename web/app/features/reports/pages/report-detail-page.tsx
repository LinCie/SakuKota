import { Link } from "@tanstack/react-router";
import { ArrowLeft, Check, Trash2 } from "lucide-react";
import { useState } from "react";

import { CategoryBadge } from "~/features/places/components/category-sign";
import { RatingStars } from "~/features/reviews/components/rating-stars";
import { ReviewMedia } from "~/features/reviews/components/review-media";
import { Button, buttonVariants } from "~/shared/components/ui/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { DetailList, type DetailItem } from "~/shared/components/detail-list";
import { MediaGallery } from "~/shared/components/media-gallery";
import { MissingSlot, orMissing } from "~/shared/components/missing-slot";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { formatLocalDate } from "~/shared/format";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { reportTarget, ReportStatusBadge } from "../components/reports-table";
import { useDismissReport, useRemoveReportedContent } from "../queries";
import {
  REASON_LABELS,
  type ReportDetail,
  type ReportedPlace,
  type ReportedReview,
} from "../types";

/** One report: the reported content as it is now, the report, and remove or dismiss. */
export function ReportDetailPage({ report }: { report: ReportDetail }) {
  const remove = useRemoveReportedContent();
  const dismiss = useDismissReport();
  // The action awaiting confirmation.
  const [confirming, setConfirming] = useState<"remove" | "dismiss" | null>(
    null,
  );
  const mutation = confirming === "dismiss" ? dismiss : remove;
  const target = reportTarget(report);
  const isReview = report.review_id !== null;
  // The content to remove is gone, so removing would only resolve the report.
  const contentGone = isReview ? report.review === null : report.place === null;

  function close() {
    setConfirming(null);
    remove.reset();
    dismiss.reset();
  }

  function confirm() {
    const kind = confirming;
    mutation.mutate(report.id, {
      onSuccess: () => {
        toastManager.add({
          title:
            kind === "remove"
              ? `Konten dihapus: ${target}`
              : `Laporan diabaikan: ${target}`,
        });
        setConfirming(null);
      },
    });
  }

  let error: string | null = null;
  if (mutation.error) {
    error = hasErrorCode(mutation.error, "ALREADY_RESOLVED")
      ? "Laporan ini sudah ditangani."
      : writeErrorMessage(mutation.error);
  }

  const details: DetailItem[] = [
    { label: "Status", value: <ReportStatusBadge status={report.status} /> },
    { label: "Alasan", value: REASON_LABELS[report.reason] },
    { label: "Pelapor", value: report.user.name },
    { label: "Dikirim", value: formatLocalDate(new Date(report.created_at)) },
  ];
  if (report.resolved_at) {
    details.push({
      label: "Ditangani",
      value: formatLocalDate(new Date(report.resolved_at)),
    });
  }
  details.push({
    label: "Catatan pelapor",
    value: orMissing(report.note, "Tanpa catatan"),
    wide: true,
  });

  return (
    <>
      <PageHeader
        back={
          <Link
            to="/admin/reports"
            className={buttonVariants({
              variant: "ghost",
              size: "sm",
              className: "text-muted-foreground -ml-3 self-start",
            })}
          >
            <ArrowLeft aria-hidden="true" data-icon="inline-start" />
            Daftar laporan
          </Link>
        }
        title={`Laporan ${isReview ? "ulasan" : "tempat"}`}
        description={target}
        actions={
          report.status === "pending" ? (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setConfirming("dismiss")}
              >
                <Check aria-hidden="true" data-icon="inline-start" />
                Abaikan
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setConfirming("remove")}
              >
                <Trash2 aria-hidden="true" data-icon="inline-start" />
                Hapus konten
              </Button>
            </>
          ) : undefined
        }
      />

      <div className="grid gap-5 lg:grid-cols-2">
        {isReview ? (
          <SectionCard title="Ulasan yang dilaporkan">
            {report.review ? (
              <ReviewContent review={report.review} />
            ) : (
              <MissingSlot>Ulasan sudah dihapus</MissingSlot>
            )}
          </SectionCard>
        ) : null}
        <SectionCard
          title={isReview ? "Tempat ulasan" : "Tempat yang dilaporkan"}
        >
          {report.place ? (
            <PlaceContent place={report.place} full={!isReview} />
          ) : (
            <MissingSlot>Tempat sudah dihapus</MissingSlot>
          )}
        </SectionCard>
        <SectionCard title="Laporan">
          <DetailList items={details} />
        </SectionCard>
      </div>

      <ConfirmDialog
        open={confirming !== null}
        onOpenChange={(open) => {
          if (!open) close();
        }}
        title={confirming === "dismiss" ? "Abaikan laporan?" : "Hapus konten?"}
        description={
          confirming === "dismiss"
            ? "Konten tetap tampil. Laporan lain tentang konten ini tetap menunggu."
            : contentGone
              ? "Konten ini sudah dihapus. Laporan ini dan laporan lain tentangnya akan ditandai selesai."
              : `${isReview ? "Ulasan ini beserta fotonya" : "Tempat ini beserta ulasan dan fotonya"} akan dihapus, dan semua laporan yang menunggu tentangnya ditandai selesai. Tindakan ini tidak bisa dibatalkan.`
        }
        confirmLabel={
          confirming === "dismiss" ? "Abaikan laporan" : "Hapus konten"
        }
        tone={confirming === "dismiss" ? "primary" : "danger"}
        pending={mutation.isPending}
        error={error}
        onConfirm={confirm}
      />
    </>
  );
}

/** The reported review: its author, stars, comment, and photos. */
function ReviewContent({ review }: { review: ReportedReview }) {
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
        <span className="text-foreground font-semibold">
          {review.user.name}
        </span>
        <RatingStars rating={review.rating} />
      </div>
      {review.comment ? (
        <p className="text-foreground text-pretty whitespace-pre-line">
          {review.comment}
        </p>
      ) : (
        <MissingSlot>Tanpa komentar</MissingSlot>
      )}
      {review.medias.length > 0 ? (
        <ul className="grid grid-cols-3 gap-2">
          {review.medias.map((media) => (
            <li key={media.id}>
              <ReviewMedia
                media={media}
                alt={`Foto ulasan ${review.user.name} ${media.position}`}
              />
            </li>
          ))}
        </ul>
      ) : null}
      <time
        dateTime={review.created_at}
        className="text-muted-foreground text-sm"
      >
        {formatLocalDate(new Date(review.created_at))}
      </time>
    </div>
  );
}

/**
 * The place: its name and category, and, when the place itself is reported
 * (`full`), its address, description, and photos.
 */
function PlaceContent({
  place,
  full,
}: {
  place: ReportedPlace;
  full: boolean;
}) {
  const items: DetailItem[] = [
    { label: "Nama", value: place.name },
    { label: "Kategori", value: <CategoryBadge category={place.category} /> },
  ];
  if (full) {
    items.push(
      {
        label: "Alamat",
        value: orMissing(place.address, "Alamat tidak tersedia"),
        wide: true,
      },
      {
        label: "Deskripsi singkat",
        value: orMissing(place.short_description, "Deskripsi tidak tersedia"),
        wide: true,
      },
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {full ? (
        <MediaGallery
          medias={place.medias}
          alt={`Foto ${place.name}`}
          missing="Foto tidak tersedia"
          aspect="aspect-4/3"
        />
      ) : null}
      <DetailList items={items} />
      <Link
        to="/admin/places/$placeId"
        params={{ placeId: String(place.id) }}
        className="text-primary self-start text-sm font-medium underline underline-offset-2"
      >
        Buka di admin tempat
      </Link>
    </div>
  );
}
