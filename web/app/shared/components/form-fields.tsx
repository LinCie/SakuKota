import type { AnyFieldApi } from "@tanstack/react-form";
import { CircleAlert } from "lucide-react";
import type { InputHTMLAttributes, ReactNode } from "react";

type TextFieldProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  "id" | "name" | "value" | "onChange" | "onBlur"
> & {
  /** The TanStack Form field this input is bound to. */
  field: AnyFieldApi;
  label: string;
  hint?: string;
};

/** A labelled paint-white input with its validation error or hint. */
export function TextField({
  field,
  label,
  hint,
  className = "",
  ...input
}: TextFieldProps) {
  // Zod errors arrive as `{ message }` objects.
  const error: string | undefined = field.state.meta.errors[0]?.message;
  const messageId = `${field.name}-message`;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={field.name} className="text-ink text-sm font-bold">
        {label}
      </label>
      <input
        id={field.name}
        name={field.name}
        value={field.state.value}
        onChange={(event) => field.handleChange(event.target.value)}
        onBlur={field.handleBlur}
        aria-invalid={error ? true : undefined}
        aria-describedby={error || hint ? messageId : undefined}
        className={`bg-paint text-ink focus-visible:outline-cat h-11 w-full rounded-[3px] border-2 px-3 focus-visible:outline-2 focus-visible:outline-offset-2 ${
          error ? "border-chili" : "border-ink"
        } ${className}`}
        {...input}
      />
      {error ? (
        <p id={messageId} className="text-chili text-sm font-semibold">
          {error}
        </p>
      ) : hint ? (
        <p id={messageId} className="text-ink-soft text-sm">
          {hint}
        </p>
      ) : null}
    </div>
  );
}

/** A form-level error message. */
export function FormAlert({ children }: { children: ReactNode }) {
  return (
    <div
      role="alert"
      className="border-chili text-ink flex items-start gap-2 rounded-[3px] border-2 px-3 py-2.5 text-sm font-semibold"
    >
      <CircleAlert
        aria-hidden="true"
        className="text-chili mt-0.5 size-4 shrink-0"
        strokeWidth={2.5}
      />
      <p>{children}</p>
    </div>
  );
}

/** The full-width yellow sign plate that submits a form. */
export function SubmitPlate({
  pending,
  children,
}: {
  pending: boolean;
  children: ReactNode;
}) {
  return (
    <button
      type="submit"
      disabled={pending}
      className="border-ink bg-sign font-sign text-ink hover:bg-sign-soft focus-visible:outline-cat disabled:bg-sign-soft mt-1 h-11 w-full rounded-[3px] border-2 px-4 text-sm focus-visible:outline-2 focus-visible:outline-offset-2"
    >
      {pending ? "Memproses…" : children}
    </button>
  );
}
