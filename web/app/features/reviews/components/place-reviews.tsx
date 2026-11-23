import { Link, useLocation } from "@tanstack/react-router";
import { lazy, Suspense, type ReactNode } from "react";

import { useSession } from "~/features/auth/session";
import { cn } from "~/shared/cn";
import { MediaCard } from "~/shared/components/media-card";
import {
  menuButtonClass,
  signButtonClass,
} from "~/shared/components/menu-board";
import { MissingSlot } from "~/shared/components/missing-slot";
import { formatLocalDate } from "~/shared/format";

import {
  useAddReviewMedias,
  useDeleteReviewMedia,
  usePlaceReviews,
} from "../queries";
import { RatingStars } from "./rating-stars";
import { ReviewMedia } from "./review-media";

// The form pulls in zod and TanStack Form, so it loads only when shown.
const ReviewForm = lazy(() =>
  import("./review-form").then((module) => ({ default: module.ReviewForm })),
);

/**
 * A place's reviews under its menu board, newest first. Signed-in users who
 * have not reviewed the place yet get a form for theirs; signed-out visitors
 * get a way to sign in and come back.
 */
export function PlaceReviews({ placeId }: { placeId: number }) {
  const session = useSession();
  const reviews = usePlaceReviews(placeId);
  const href = useLocation({ select: (location) => location.href });
  // Here, not in the form or the card, so files picked in the form keep
  // their progress and errors once the saved review swaps the form for the card.
  const addMedias = useAddReviewMedias(placeId);
  const deleteMedia = useDeleteReviewMedia(placeId);

  let writer: ReactNode = null;
  if (!session) {
    writer = (
      <Link
        to="/masuk"
        search={{ kembali: href === "/" ? undefined : href }}
        className={cn(
          signButtonClass,
          "inline-flex h-11 items-center justify-center",
        )}
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
        <p className="text-ink-soft text-sm">Kamu sudah mengulas tempat ini.</p>
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
    list = <p className="text-ink-soft text-sm">Memuat ulasan…</p>;
  } else if (reviews.isError) {
    list = (
      <p className="text-ink-soft flex items-center gap-2 text-sm">
        Ulasan belum bisa dimuat.
        <button
          type="button"
          onClick={() => void reviews.refetch()}
          className={cn(
            menuButtonClass,
            "rounded-[3px] px-2 py-1 font-semibold",
          )}
        >
          Coba lagi
        </button>
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
          <li
            key={review.id}
            className="border-ink/25 flex flex-col gap-1 border-t border-dashed py-3"
          >
            <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
              <span className="text-ink font-semibold">{review.user.name}</span>
              <RatingStars rating={review.rating} />
            </div>
            {review.comment ? (
              <p className="text-ink text-pretty whitespace-pre-line">
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
            <time
              dateTime={review.created_at}
              className="text-ink-soft text-sm"
            >
              {formatLocalDate(new Date(review.created_at))}
            </time>
          </li>
        ))}
      </ul>
    );
  }

  return (
    <section
      aria-labelledby="place-reviews"
      className="flex flex-col gap-3 px-3 pb-6 md:px-4"
    >
      <h3 id="place-reviews" className="font-sign text-ink text-base">
        Ulasan
      </h3>
      {writer}
      {list}
    </section>
  );
}
