import { useForm, type AnyFieldApi } from "@tanstack/react-form";
import { Star } from "lucide-react";

import { cn } from "~/shared/cn";
import {
  controlClass,
  FieldLabel,
  FieldMessage,
  TextAreaField,
} from "~/shared/components/field";
import { FormAlert } from "~/shared/components/form-alert";
import { SubmitPlate } from "~/shared/components/submit-plate";
import { imageAccept, maxMedias } from "~/shared/image-upload";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { useCreateReview } from "../queries";
import { reviewSchema, type ReviewValues } from "../schemas";
import { STARS } from "./rating-stars";

const emptyReview: ReviewValues = { rating: 0, comment: "", files: [] };

type ReviewFormProps = {
  placeId: number;
  /** Called with the picked files once the review is saved, to add them to it. */
  onCreated: (files: File[]) => void;
};

/** The signed-in user's review of a place: a star rating, an optional comment, and optional files. */
export function ReviewForm({ placeId, onCreated }: ReviewFormProps) {
  const createReview = useCreateReview(placeId);

  const form = useForm({
    defaultValues: emptyReview,
    validators: { onSubmit: reviewSchema },
    onSubmit: ({ value }) => {
      // mutateAsync, not mutate with callbacks: the saved review replaces this
      // form, and callbacks of an unmounted component's mutate never run.
      createReview
        .mutateAsync({
          rating: value.rating,
          comment: value.comment.trim() || null,
        })
        .then(
          () => onCreated(value.files),
          // Shown from createReview.error.
          () => {},
        );
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
      <form.Field name="files">
        {(field) => <FilesField field={field} />}
      </form.Field>

      <SubmitPlate pending={createReview.isPending}>Kirim ulasan</SubmitPlate>
    </form>
  );
}

/** A native picker for up to `maxMedias` images or videos. */
function FilesField({ field }: { field: AnyFieldApi }) {
  // Zod errors arrive as `{ message }` objects.
  const error: string | undefined = field.state.meta.errors[0]?.message;
  const messageId = `${field.name}-message`;

  return (
    <div className="flex flex-col gap-1.5">
      <FieldLabel htmlFor={field.name} optional>
        Foto atau video
      </FieldLabel>
      <input
        id={field.name}
        name={field.name}
        type="file"
        accept={imageAccept}
        multiple
        onChange={(event) =>
          field.handleChange(Array.from(event.target.files ?? []))
        }
        onBlur={field.handleBlur}
        aria-invalid={error ? true : undefined}
        aria-describedby={messageId}
        className={cn(
          controlClass(Boolean(error)),
          "file:text-ink py-2 text-sm file:mr-3 file:font-semibold",
        )}
      />
      <FieldMessage
        id={messageId}
        error={error}
        hint={`Maks. ${maxMedias} file. Gambar JPEG, PNG, atau WebP (maks. 5 MB); video MP4 atau WebM (maks. 50 MB).`}
      />
    </div>
  );
}

/** Five star radios; picking a star rates the place that many stars. */
function StarField({ field }: { field: AnyFieldApi }) {
  // Zod errors arrive as `{ message }` objects.
  const error: string | undefined = field.state.meta.errors[0]?.message;
  const messageId = `${field.name}-message`;
  const value = field.state.value as number;

  // Stars up to the picked one are painted; the rest turn red on an error.
  function starClass(star: number) {
    if (star <= value) return "fill-sign text-ink";
    if (error) return "text-chili";
    return "text-ink/40";
  }

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
              className={cn("size-7", starClass(star))}
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
