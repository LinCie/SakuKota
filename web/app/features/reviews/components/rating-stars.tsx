import { Star } from "lucide-react";

import { cn } from "~/shared/cn";

export const STARS = [1, 2, 3, 4, 5] as const;

/** A rating as five stars, the first `rating` of them filled kunyit. */
export function RatingStars({
  rating,
  className,
}: {
  rating: number;
  className?: string;
}) {
  return (
    <span
      role="img"
      aria-label={`${rating} dari 5 bintang`}
      className={cn("inline-flex shrink-0 gap-0.5", className)}
    >
      {STARS.map((star) => (
        <Star
          key={star}
          aria-hidden="true"
          className={cn(
            "size-4",
            star <= rating ? "fill-kunyit text-kunyit" : "text-input",
          )}
        />
      ))}
    </span>
  );
}
