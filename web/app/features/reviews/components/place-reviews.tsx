import { Link, useLocation } from "@tanstack/react-router";
import { lazy, Suspense, type ReactNode } from "react";

import { useSession } from "~/features/auth/session";
import { cn } from "~/shared/cn";
import {
  menuButtonClass,
  signButtonClass,
} from "~/shared/components/menu-board";
import { MissingSlot } from "~/shared/components/missing-slot";
import { formatLocalDate } from "~/shared/format";

import { usePlaceReviews } from "../queries";
import { RatingStars } from "./rating-stars";

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
    const reviewed = reviews.data.some(
      (review) => review.user.id === session.user.id,
    );
    writer = reviewed ? (
      <p className="text-ink-soft text-sm">Kamu sudah mengulas tempat ini.</p>
    ) : (
      <Suspense fallback={null}>
        <ReviewForm placeId={placeId} />
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
