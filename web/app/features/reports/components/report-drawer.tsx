import { useForm, useSelector } from "@tanstack/react-form";

import { SelectField, TextAreaField } from "~/shared/components/field";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { orNull } from "~/shared/text";
import { toastManager } from "~/shared/toast-manager";
import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

import { useCreateReport } from "../queries";
import { reportSchema, type ReportValues } from "../schemas";
import { REASON_LABELS, REPORT_REASONS, type ReportReason } from "../types";

const FORM_ID = "report-form";

const emptyReport: ReportValues = { reason: "", note: "" };

const reasonOptions = REPORT_REASONS.map((reason) => ({
  value: reason,
  label: REASON_LABELS[reason],
}));

type ReportDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  placeId: number;
  /** The reported review; null reports the place itself. */
  reviewId: number | null;
  /** Names the content mid-sentence, e.g. "ulasan Ana" or "tempat Warung Bu Tini". */
  target: string;
};

/** Reports a place or a review to the admins, with a reason and an optional note. */
export function ReportDrawer({
  open,
  onOpenChange,
  placeId,
  reviewId,
  target,
}: ReportDrawerProps) {
  const create = useCreateReport(placeId, reviewId);

  const form = useForm({
    defaultValues: emptyReport,
    validators: { onSubmit: reportSchema },
    onSubmit: ({ value }) => {
      create.mutate(
        {
          // The schema checked it is one of the reasons.
          reason: value.reason as ReportReason,
          note: orNull(value.note),
        },
        {
          onSuccess: () => {
            toastManager.add({ title: "Laporan terkirim. Terima kasih." });
            onOpenChange(false);
          },
        },
      );
    },
  });

  const dirty = useSelector(form.store, (state) => state.isDirty);

  let error: string | null = null;
  if (create.error) {
    if (hasErrorCode(create.error, "ALREADY_REPORTED")) {
      error = "Kamu sudah melaporkan ini dan laporannya masih ditinjau.";
    } else if (hasErrorCode(create.error, "TOO_MANY_PENDING_REPORTS")) {
      error =
        "Kamu punya 5 laporan yang masih ditinjau. Tunggu admin menanganinya dulu.";
    } else if (
      hasErrorCode(create.error, "PLACE_NOT_FOUND") ||
      hasErrorCode(create.error, "REVIEW_NOT_FOUND")
    ) {
      error = "Konten ini sudah tidak ada.";
    } else {
      error = writeErrorMessage(create.error);
    }
  }

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={() => {
        form.reset(emptyReport);
        create.reset();
      }}
      title={reviewId === null ? "Laporkan tempat" : "Laporkan ulasan"}
      description={`Laporanmu tentang ${target} akan ditinjau admin. Namamu tidak ditampilkan ke publik.`}
      dirty={dirty}
      formId={FORM_ID}
      pending={create.isPending}
      submitLabel="Kirim laporan"
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
        {error ? <FormAlert>{error}</FormAlert> : null}
        <form.Field name="reason">
          {(field) => (
            <SelectField
              field={field}
              label="Alasan"
              placeholder="Pilih alasan"
              options={reasonOptions}
            />
          )}
        </form.Field>
        <form.Field name="note">
          {(field) => (
            <TextAreaField
              field={field}
              label="Catatan"
              optional
              hint="Jelaskan apa yang salah agar admin cepat menanganinya."
            />
          )}
        </form.Field>
      </form>
    </FormDrawer>
  );
}
