import { useForm, type AnyFieldApi } from "@tanstack/react-form";
import { Star } from "lucide-react";

import { cn } from "~/shared/cn";
import { FieldMessage, TextAreaField } from "~/shared/components/field";
import { FormAlert } from "~/shared/components/form-alert";
import { SubmitPlate } from "~/shared/components/submit-plate";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { useCreateReview } from "../queries";
import { reviewSchema, type ReviewValues } from "../schemas";
import { STARS } from "./rating-stars";

const emptyReview: ReviewValues = { rating: 0, comment: "" };

/** The signed-in user's review of a place: a star rating and an optional comment. */
export function ReviewForm({ placeId }: { placeId: number }) {
  const createReview = useCreateReview(placeId);

  const form = useForm({
    defaultValues: emptyReview,
    validators: { onSubmit: reviewSchema },
    onSubmit: ({ value }) => {
      createReview.mutate({
        rating: value.rating,
        comment: value.comment.trim() || null,
      });
    },
  });

  return (
    <form
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
      className="flex flex-col gap-4"
    >
      {createReview.error ? (
        <FormAlert>
          {hasErrorCode(createReview.error, "ALREADY_REVIEWED")
            ? "Kamu sudah mengulas tempat ini."
            : writeErrorMessage(createReview.error)}
        </FormAlert>
      ) : null}

      <form.Field name="rating">
        {(field) => <StarField field={field} />}
      </form.Field>
      <form.Field name="comment">
        {(field) => (
          <TextAreaField
            field={field}
            label="Komentar"
            optional
            hint="Ceritakan harga, pelayanan, atau hal lain yang perlu diketahui."
          />
        )}
      </form.Field>

      <SubmitPlate pending={createReview.isPending}>Kirim ulasan</SubmitPlate>
    </form>
  );
}

/** Five star radios; picking a star rates the place that many stars. */
function StarField({ field }: { field: AnyFieldApi }) {
  // Zod errors arrive as `{ message }` objects.
  const error: string | undefined = field.state.meta.errors[0]?.message;
  const messageId = `${field.name}-message`;
  const value = field.state.value as number;

  return (
    <fieldset aria-describedby={error ? messageId : undefined}>
      <legend className="text-ink mb-1.5 text-sm font-bold">Rating</legend>
      <div className="flex">
        {STARS.map((star) => (
          <label
            key={star}
            className="hover:bg-sign-soft has-focus-visible:outline-cat relative grid size-11 cursor-pointer place-items-center rounded-[3px] transition-colors has-focus-visible:outline-2"
          >
            <input
              type="radio"
              name={field.name}
              value={star}
              checked={value === star}
              onChange={() => field.handleChange(star)}
              onBlur={field.handleBlur}
              className="sr-only"
            />
            <Star
              aria-hidden="true"
              strokeWidth={2.5}
              className={cn(
                "size-7",
                star <= value
                  ? "fill-sign text-ink"
                  : error
                    ? "text-chili"
                    : "text-ink/40",
              )}
            />
            <span className="sr-only">{star} bintang</span>
          </label>
        ))}
      </div>
      <div className="mt-1.5">
        <FieldMessage id={messageId} error={error} />
      </div>
    </fieldset>
  );
}
