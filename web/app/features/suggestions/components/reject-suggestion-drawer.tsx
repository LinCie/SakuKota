import { useForm, useSelector } from "@tanstack/react-form";

import { TextAreaField } from "~/shared/components/field";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { orNull } from "~/shared/text";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { useRejectSuggestion } from "../queries";
import { rejectSchema } from "../schemas";
import type { Suggestion } from "../types";

const FORM_ID = "reject-suggestion-form";

const emptyReject = { reason: "" };

type RejectSuggestionDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  suggestion: Suggestion;
};

/** Rejects `suggestion`, with an optional reason the user gets to read. */
export function RejectSuggestionDrawer({
  open,
  onOpenChange,
  suggestion,
}: RejectSuggestionDrawerProps) {
  const reject = useRejectSuggestion();

  const form = useForm({
    defaultValues: emptyReject,
    validators: { onSubmit: rejectSchema },
    onSubmit: ({ value }) => {
      reject.mutate(
        { id: suggestion.id, body: { reason: orNull(value.reason) } },
        {
          onSuccess: () => {
            toastManager.add({ title: `Usulan ditolak: ${suggestion.name}` });
            onOpenChange(false);
          },
        },
      );
    },
  });

  const dirty = useSelector(form.store, (state) => state.isDirty);

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(emptyReject);
        reject.reset();
      }}
      title="Tolak usulan"
      description={`${suggestion.name} tidak akan masuk ke peta. Pengusul bisa membaca alasannya.`}
      dirty={dirty}
      formId={FORM_ID}
      pending={reject.isPending}
      submitLabel="Tolak usulan"
    >
      <form
        id={FORM_ID}
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
        className="flex flex-col gap-4"
      >
        {reject.error ? (
          <FormAlert>
            {hasErrorCode(reject.error, "ALREADY_REVIEWED")
              ? "Usulan ini sudah ditinjau."
              : writeErrorMessage(reject.error)}
          </FormAlert>
        ) : null}
        <form.Field name="reason">
          {(field) => (
            <TextAreaField
              field={field}
              label="Alasan"
              optional
              hint="Contoh: tempat ini sudah ada di peta."
            />
          )}
        </form.Field>
      </form>
    </FormDrawer>
  );
}
