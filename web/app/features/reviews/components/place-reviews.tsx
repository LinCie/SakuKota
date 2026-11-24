import { Link, useLocation } from "@tanstack/react-router";
import { lazy, Suspense, useState, type ReactNode } from "react";

import { isAdmin, useSession } from "~/features/auth/session";
import { ReportButton } from "~/features/reports/components/report-button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { MediaCard } from "~/shared/components/media-card";
import { MissingSlot } from "~/shared/components/missing-slot";
import { Button, buttonVariants } from "~/shared/components/ui/button";
import { formatLocalDate } from "~/shared/format";
import { writeErrorMessage } from "~/shared/write-error";

import {
  useAddReviewMedias,
  useDeleteReview,
  useDeleteReviewMedia,
  usePlaceReviews,
} from "../queries";
import type { Review } from "../types";
import { RatingStars } from "./rating-stars";
import { ReviewMedia } from "./review-media";

// The form pulls in zod and TanStack Form, so it loads only when shown.
const ReviewForm = lazy(() =>
  import("./review-form").then((module) => ({ default: module.ReviewForm })),
);

/**
 * A place's reviews under its details, newest first. Signed-in users who
 * have not reviewed the place yet get a form for theirs; signed-out visitors
 * get a way to sign in and come back. Users can delete their own review and
 * report the others; admins can delete any review.
 */
export function PlaceReviews({ placeId }: { placeId: number }) {
  const session = useSession();
  const reviews = usePlaceReviews(placeId);
  const href = useLocation({ select: (location) => location.href });
  // Here, not in the form or the card, so files picked in the form keep
  // their progress and errors once the saved review swaps the form for the card.
  const addMedias = useAddReviewMedias(placeId);
  const deleteMedia = useDeleteReviewMedia(placeId);
  const admin = isAdmin(session);
  const deleteReview = useDeleteReview(placeId);
  // The review asked to be deleted, while the confirmation is open.
  const [toDelete, setToDelete] = useState<Review | null>(null);
  const isOwn = (review: Review) => review.user.id === session?.user.id;

  let writer: ReactNode = null;
  if (!session) {
    writer = (
      <Link
        to="/masuk"
        search={{ kembali: href === "/" ? undefined : href }}
        className={buttonVariants({ variant: "secondary" })}
      >
        Masuk untuk menulis ulasan
      </Link>
    );
  } else if (reviews.isSuccess) {
    const own = reviews.data.find(
      (review) => review.user.id === session.user.id,
    );
    writer = own ? (
      <>
        <p className="text-muted-foreground text-sm">
          Kamu sudah mengulas tempat ini.
        </p>
        <MediaCard
          title="Foto ulasanmu"
          medias={own.medias}
          alt="Foto ulasanmu"
          missing="Belum ada foto"
          aspect="aspect-square"
          pending={addMedias.isPending}
          deletingId={deleteMedia.isPending ? deleteMedia.variables : null}
          error={addMedias.error ?? deleteMedia.error}
          onAdd={(file) => {
            deleteMedia.reset();
            addMedias.mutate([file]);
          }}
          onDelete={(media) => {
            addMedias.reset();
            deleteMedia.mutate(media.id);
          }}
        />
      </>
    ) : (
      <Suspense fallback={null}>
        <ReviewForm
          placeId={placeId}
          onCreated={(files) => {
            if (files.length > 0) addMedias.mutate(files);
          }}
        />
      </Suspense>
    );
  }

  let list: ReactNode;
  if (reviews.isPending) {
    list = <p className="text-muted-foreground text-sm">Memuat ulasan…</p>;
  } else if (reviews.isError) {
    list = (
      <p className="text-muted-foreground flex items-center gap-2 text-sm">
        Ulasan belum bisa dimuat.
        <Button
          variant="ghost"
          size="sm"
          onClick={() => void reviews.refetch()}
          className="text-foreground"
        >
          Coba lagi
        </Button>
      </p>
    );
  } else if (reviews.data.length === 0) {
    list = (
      <div>
        <MissingSlot>Belum ada ulasan</MissingSlot>
      </div>
    );
  } else {
    list = (
      <ul>
        {reviews.data.map((review) => (
          <li key={review.id} className="flex flex-col gap-1.5 border-t py-3">
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
            ) : null}
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
            <div className="flex items-center justify-between gap-3">
              <time
                dateTime={review.created_at}
                className="text-muted-foreground text-xs"
              >
                {formatLocalDate(new Date(review.created_at))}
              </time>
              {admin || isOwn(review) ? (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setToDelete(review)}
                  aria-label={`Hapus ulasan ${review.user.name}`}
                  className="text-destructive hover:text-destructive"
                >
                  Hapus
                </Button>
              ) : (
                <ReportButton
                  placeId={placeId}
                  reviewId={review.id}
                  target={`ulasan ${review.user.name}`}
                />
              )}
            </div>
          </li>
        ))}
      </ul>
    );
  }

  return (
    <section
      aria-labelledby="place-reviews"
      className="flex flex-col gap-3 px-4 pb-6 md:px-5"
    >
      <h3
        id="place-reviews"
        className="text-foreground text-base font-semibold"
      >
        Ulasan
      </h3>
      {writer}
      {list}
      {session ? (
        <ConfirmDialog
          open={toDelete !== null}
          onOpenChange={(open) => {
            if (open) return;
            setToDelete(null);
            deleteReview.reset();
          }}
          title="Hapus ulasan?"
          description={`${toDelete && !isOwn(toDelete) ? `Ulasan ${toDelete.user.name}` : "Ulasanmu"} beserta fotonya akan dihapus dan ratingnya tidak dihitung lagi. Tindakan ini tidak bisa dibatalkan.`}
          confirmLabel="Hapus ulasan"
          pending={deleteReview.isPending}
          error={
            deleteReview.error ? writeErrorMessage(deleteReview.error) : null
          }
          onConfirm={() => {
            if (toDelete)
              // Own reviews go through "mine", which needs no admin role.
              deleteReview.mutate(isOwn(toDelete) ? "mine" : toDelete.id, {
                onSuccess: () => setToDelete(null),
              });
          }}
        />
      ) : null}
    </section>
  );
}
