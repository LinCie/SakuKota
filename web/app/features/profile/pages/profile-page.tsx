import { useForm } from "@tanstack/react-form";
import { Link } from "@tanstack/react-router";
import {
  ArrowLeft,
  Camera,
  CalendarDays,
  LoaderCircle,
  MapPin,
  Star,
  X,
} from "lucide-react";
import { useRef, useState } from "react";

import { AccountControl } from "~/features/auth/components/account-control";
import { RatingStars, STARS } from "~/features/reviews/components/rating-stars";
import { ReviewMedia } from "~/features/reviews/components/review-media";
import { useMyReviews } from "~/features/reviews/queries";
import type { Review } from "~/features/reviews/types";
import { cn } from "~/shared/cn";
import { ErrorBoard } from "~/shared/components/error-board";
import { TextField } from "~/shared/components/field";
import { FormAlert } from "~/shared/components/form-alert";
import { MissingSlot } from "~/shared/components/missing-slot";
import { SectionCard } from "~/shared/components/section-card";
import { SubmitPlate } from "~/shared/components/submit-plate";
import { SiteHeader } from "~/shared/components/site-header";
import { Toaster } from "~/shared/components/toaster";
import { Badge } from "~/shared/components/ui/badge";
import { Button, buttonVariants } from "~/shared/components/ui/button";
import { UserAvatar } from "~/shared/components/user-avatar";

import {
  formatLocalDate,
  formatLocalMonth,
  formatNumber,
  formatRating,
} from "~/shared/format";
import { imageErrorMessage } from "~/shared/image-upload";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import { avatarAccept, useProfile, useRename, useSetAvatar } from "../queries";
import { profileSchema } from "../schemas";

/** How many reviews show at first, and how many more each "show more" adds. */
const reviewPage = 20;

/** Where a signed-in user sets their photo and display name and sees their reviews. */
export function ProfilePage({ userId }: { userId: number }) {
  return (
    <Toaster>
      <div className="bg-background text-foreground flex min-h-dvh flex-col">
        <SiteHeader>
          <AccountControl />
        </SiteHeader>

        <main className="mx-auto grid w-full max-w-5xl gap-8 px-3 py-6 md:px-5 lg:grid-cols-[20rem_minmax(0,1fr)] lg:items-start lg:gap-10">
          <div className="flex flex-col gap-6 lg:sticky lg:top-6">
            <Link
              to="/"
              className={cn(
                buttonVariants({ variant: "ghost" }),
                "text-muted-foreground -ml-3 self-start",
              )}
            >
              <ArrowLeft aria-hidden="true" data-icon="inline-start" />
              Kembali ke peta
            </Link>
            <Identity userId={userId} />
            <NameCard userId={userId} />
          </div>
          <MyReviews userId={userId} />
        </main>
      </div>
    </Toaster>
  );
}

/** The user's photo, with the button that changes it, over their name, email, and join date. */
function Identity({ userId }: { userId: number }) {
  const { data: profile } = useProfile(userId);
  const setAvatar = useSetAvatar();
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-4 lg:flex-col lg:items-start">
        <div className="relative shrink-0">
          <UserAvatar
            name={profile.name}
            url={profile.avatar_url}
            alt="Foto profilmu"
            className={cn(
              "ring-card size-20 text-3xl ring-4 transition-opacity lg:size-28 lg:text-4xl",
              setAvatar.isPending && "opacity-60",
            )}
          />
          <Button
            variant="outline"
            size="icon-sm"
            aria-label={
              setAvatar.isPending
                ? "Mengunggah foto…"
                : profile.avatar_url
                  ? "Ganti foto"
                  : "Tambah foto"
            }
            aria-describedby="avatar-hint"
            disabled={setAvatar.isPending}
            onClick={() => inputRef.current?.click()}
            className="ring-background absolute -right-1 -bottom-1 ring-4 disabled:opacity-100"
          >
            {setAvatar.isPending ? (
              <LoaderCircle
                aria-hidden="true"
                className="motion-safe:animate-spin"
              />
            ) : (
              <Camera aria-hidden="true" />
            )}
          </Button>
          <input
            ref={inputRef}
            type="file"
            accept={avatarAccept}
            hidden
            onChange={(event) => {
              const file = event.target.files?.[0];
              // Cleared so picking the same file again still uploads it.
              event.target.value = "";
              if (file) setAvatar.mutate(file);
            }}
          />
        </div>

        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <h1 className="text-foreground text-2xl leading-tight font-semibold tracking-[-0.02em] text-balance wrap-break-word">
              {profile.name}
            </h1>
            {profile.role === "admin" ? (
              <Badge variant="outline">Admin</Badge>
            ) : null}
          </div>
          <p className="text-muted-foreground truncate text-sm">
            {profile.email}
          </p>
          <p className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
            <CalendarDays aria-hidden="true" className="size-3.5" />
            Bergabung {formatLocalMonth(new Date(profile.created_at))}
          </p>
        </div>
      </div>

      <p id="avatar-hint" className="text-muted-foreground text-xs">
        Foto: JPEG, PNG, atau WebP (maks. 5 MB).
      </p>
      {setAvatar.error ? (
        <FormAlert>{imageErrorMessage(setAvatar.error)}</FormAlert>
      ) : null}
    </div>
  );
}

/** The display name, which the user's reviews show. */
function NameCard({ userId }: { userId: number }) {
  const { data: profile } = useProfile(userId);
  const rename = useRename();

  const form = useForm({
    defaultValues: { name: profile.name },
    validators: { onSubmit: profileSchema },
    onSubmit: ({ value }) =>
      rename.mutateAsync(value.name).then(
        () => toastManager.add({ title: "Nama disimpan" }),
        // Shown from rename.error.
        () => {},
      ),
  });

  return (
    <SectionCard title="Ubah nama">
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
        className="flex flex-col gap-4"
      >
        {rename.error ? (
          <FormAlert>{writeErrorMessage(rename.error)}</FormAlert>
        ) : null}
        <form.Field name="name">
          {(field) => (
            <TextField
              field={field}
              label="Nama tampilan"
              autoComplete="name"
              hint="Tampil di setiap ulasanmu."
            />
          )}
        </form.Field>
        <SubmitPlate pending={rename.isPending}>Simpan nama</SubmitPlate>
      </form>
    </SectionCard>
  );
}

/** The user's reviews, newest first and grouped by month, each linking to its place on the map. */
function MyReviews({ userId }: { userId: number }) {
  const { data: reviews } = useMyReviews(userId);
  // The star count the list is narrowed to, or null for every review.
  const [stars, setStars] = useState<number | null>(null);
  const [visible, setVisible] = useState(reviewPage);

  function filterBy(star: number | null) {
    setStars(star);
    setVisible(reviewPage);
  }

  const shown =
    stars === null
      ? reviews
      : reviews.filter((review) => review.rating === stars);

  return (
    <section aria-labelledby="ulasanku" className="flex min-w-0 flex-col gap-5">
      <div className="flex items-baseline justify-between gap-3">
        <h2
          id="ulasanku"
          className="text-foreground text-xl font-semibold tracking-[-0.01em]"
        >
          Ulasanku
        </h2>
        {reviews.length > 0 ? (
          <p className="text-muted-foreground text-sm tabular-nums">
            {formatNumber(reviews.length)} ulasan
          </p>
        ) : null}
      </div>

      {reviews.length === 0 ? (
        <NoReviews />
      ) : (
        <>
          <StarSummary reviews={reviews} stars={stars} onPick={filterBy} />

          {stars !== null ? (
            <div className="flex items-center justify-between gap-3">
              <p className="text-muted-foreground text-sm" aria-live="polite">
                <span className="text-foreground font-semibold tabular-nums">
                  {formatNumber(shown.length)}
                </span>{" "}
                ulasan bintang {stars}
              </p>
              <Button variant="ghost" size="sm" onClick={() => filterBy(null)}>
                <X aria-hidden="true" data-icon="inline-start" />
                Tampilkan semua
              </Button>
            </div>
          ) : null}

          <ReviewMonths reviews={shown.slice(0, visible)} />

          {shown.length > visible ? (
            <Button
              variant="outline"
              className="self-center"
              onClick={() => setVisible(visible + reviewPage)}
            >
              Tampilkan lebih banyak
              <span className="text-muted-foreground tabular-nums">
                ({formatNumber(shown.length - visible)} lagi)
              </span>
            </Button>
          ) : null}
        </>
      )}
    </section>
  );
}

/** The average star the user gives and how their reviews spread over 1 to 5 stars; each row narrows the list to that star. */
function StarSummary({
  reviews,
  stars,
  onPick,
}: {
  reviews: Review[];
  stars: number | null;
  onPick: (star: number | null) => void;
}) {
  const counts = new Map<number, number>();
  let total = 0;
  for (const review of reviews) {
    counts.set(review.rating, (counts.get(review.rating) ?? 0) + 1);
    total += review.rating;
  }
  const photos = reviews.reduce((sum, review) => sum + review.medias.length, 0);

  return (
    <div className="bg-card ring-foreground/10 grid gap-5 rounded-xl p-5 ring-1 sm:grid-cols-[auto_minmax(0,1fr)] sm:gap-8">
      <div className="flex flex-col gap-1 sm:justify-center sm:border-r sm:pr-8">
        <p className="flex items-center gap-2">
          <Star aria-hidden="true" className="fill-kunyit text-kunyit size-6" />
          <span className="text-foreground text-[1.75rem] leading-none font-semibold tracking-[-0.02em] tabular-nums">
            {formatRating(total / reviews.length)}
          </span>
        </p>
        <p className="text-muted-foreground text-sm">
          rata-rata bintang yang kamu beri
        </p>
        {photos > 0 ? (
          <p className="text-muted-foreground pt-2 text-sm">
            <span className="text-foreground font-semibold tabular-nums">
              {formatNumber(photos)}
            </span>{" "}
            foto dibagikan
          </p>
        ) : null}
      </div>

      <div
        role="group"
        aria-label="Saring ulasan menurut bintang"
        className="-mx-2 flex flex-col gap-0.5"
      >
        {[...STARS].reverse().map((star) => {
          const count = counts.get(star) ?? 0;
          const picked = stars === star;
          return (
            <button
              key={star}
              type="button"
              aria-pressed={picked}
              aria-label={`Ulasan bintang ${star}: ${count}`}
              disabled={count === 0}
              onClick={() => onPick(picked ? null : star)}
              className="hover:bg-muted focus-visible:ring-ring/50 aria-pressed:bg-pandan-soft aria-pressed:text-pandan-deep grid h-9 grid-cols-[2rem_minmax(0,1fr)_2.5rem] items-center gap-3 rounded-full px-3 text-sm font-medium transition-colors outline-none focus-visible:ring-3 disabled:pointer-events-none disabled:opacity-50"
            >
              <span className="flex items-center gap-1 tabular-nums">
                {star}
                <Star
                  aria-hidden="true"
                  className="fill-kunyit text-kunyit size-3.5"
                />
              </span>
              <progress
                aria-hidden="true"
                value={count}
                max={reviews.length}
                className="bg-muted [&::-moz-progress-bar]:bg-kunyit [&::-webkit-progress-bar]:bg-muted [&::-webkit-progress-value]:bg-kunyit h-2 w-full appearance-none overflow-hidden rounded-full [&::-moz-progress-bar]:rounded-full [&::-webkit-progress-value]:rounded-full"
              />
              <span className="text-right tabular-nums">
                {formatNumber(count)}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

/** Reviews in one card per month, newest month first, in the order given. */
function ReviewMonths({ reviews }: { reviews: Review[] }) {
  const months: { key: string; label: string; reviews: Review[] }[] = [];
  for (const review of reviews) {
    const date = new Date(review.created_at);
    const key = `${date.getFullYear()}-${date.getMonth()}`;
    const last = months.at(-1);
    if (last?.key === key) {
      last.reviews.push(review);
    } else {
      months.push({ key, label: formatLocalMonth(date), reviews: [review] });
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {months.map((month) => (
        <section
          key={month.key}
          aria-label={month.label}
          className="flex flex-col gap-2"
        >
          <h3 className="text-muted-foreground flex items-baseline justify-between gap-3 px-1 text-sm font-medium">
            {month.label}
            <span className="text-xs tabular-nums">
              {formatNumber(month.reviews.length)} ulasan
            </span>
          </h3>
          <ul className="bg-card ring-foreground/10 rounded-xl ring-1">
            {month.reviews.map((review) => (
              <ReviewRow key={review.id} review={review} />
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

function ReviewRow({ review }: { review: Review }) {
  return (
    <li className="flex flex-col gap-2 border-t px-5 py-4 first:border-t-0">
      <div className="flex flex-wrap items-start justify-between gap-x-3 gap-y-1">
        <span className="text-foreground font-semibold text-pretty">
          {review.place_name}
        </span>
        <RatingStars rating={review.rating} className="mt-1" />
      </div>
      {review.comment ? (
        <p className="text-foreground max-w-[65ch] text-pretty whitespace-pre-line">
          {review.comment}
        </p>
      ) : null}
      {review.medias.length > 0 ? (
        <ul className="grid grid-cols-[repeat(auto-fill,minmax(4.5rem,1fr))] gap-2 sm:grid-cols-[repeat(auto-fill,5rem)]">
          {review.medias.map((media) => (
            <li key={media.id}>
              <ReviewMedia
                media={media}
                alt={`Foto ulasan ${review.place_name} ${media.position}`}
              />
            </li>
          ))}
        </ul>
      ) : null}
      <div className="-mb-1.5 flex items-center justify-between gap-3">
        <time
          dateTime={review.created_at}
          className="text-muted-foreground text-xs"
        >
          {formatLocalDate(new Date(review.created_at))}
        </time>
        <Link
          to="/"
          search={{ tempat: review.place_id }}
          className={cn(
            buttonVariants({ variant: "ghost", size: "sm" }),
            "text-primary hover:text-primary -mr-3",
          )}
        >
          <MapPin aria-hidden="true" data-icon="inline-start" />
          Lihat di peta
        </Link>
      </div>
    </li>
  );
}

/** Shown before the user's first review: says where reviews come from. */
function NoReviews() {
  return (
    <div className="bg-card ring-foreground/10 flex flex-col items-start gap-3 rounded-xl p-5 ring-1">
      <MissingSlot>Belum ada ulasan</MissingSlot>
      <p className="text-muted-foreground max-w-[65ch] text-sm text-pretty">
        Buka sebuah tempat di peta untuk menulis ulasan. Semua ulasanmu akan
        terkumpul di sini.
      </p>
      <Link to="/" className={buttonVariants({ variant: "outline" })}>
        <MapPin aria-hidden="true" data-icon="inline-start" />
        Buka peta
      </Link>
    </div>
  );
}

/** Shown when the profile page's data fails to load. */
export function ProfileError() {
  return <ErrorBoard title="Profil belum bisa dimuat." href="/profil" />;
}
