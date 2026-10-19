import type { AnyFieldApi } from "@tanstack/react-form";
import { ChevronDown, CircleAlert } from "lucide-react";
import type {
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from "react";

import { cn } from "~/shared/cn";
import { signButtonClass } from "./menu-board";

/** Shared look of a paint-white control; the border turns chili when invalid. */
export function controlClass(invalid: boolean) {
  return cn(
    "bg-paint text-ink focus-visible:outline-cat w-full rounded-[3px] border-2 px-3 focus-visible:outline-2 focus-visible:outline-offset-2",
    invalid ? "border-chili" : "border-ink",
  );
}

/** A text value as a nullable string: blank becomes null. */
export function orNull(text: string) {
  const trimmed = text.trim();
  return trimmed === "" ? null : trimmed;
}

/** A field's label; optional fields say so, so required ones need no mark. */
export function FieldLabel({
  htmlFor,
  optional,
  children,
}: {
  htmlFor?: string;
  optional?: boolean;
  children: ReactNode;
}) {
  return (
    <label htmlFor={htmlFor} className="text-ink text-sm font-bold">
      {children}
      {optional ? (
        <span className="text-ink-soft ml-1.5 font-normal">opsional</span>
      ) : null}
    </label>
  );
}

/** The error (chili) or hint (soft ink) line under a field. */
export function FieldMessage({
  id,
  error,
  hint,
}: {
  id: string;
  error?: string;
  hint?: string;
}) {
  if (error) {
    return (
      <p id={id} className="text-chili text-sm font-semibold">
        {error}
      </p>
    );
  }
  return hint ? (
    <p id={id} className="text-ink-soft text-sm">
      {hint}
    </p>
  ) : null;
}

// Zod errors arrive as `{ message }` objects.
function fieldError(field: AnyFieldApi): string | undefined {
  return field.state.meta.errors[0]?.message;
}

type TextFieldProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  "id" | "name" | "value" | "onChange" | "onBlur" | "prefix"
> & {
  /** The TanStack Form field this input is bound to. */
  field: AnyFieldApi;
  label: string;
  hint?: string;
  optional?: boolean;
  /** A fixed lead-in inside the input, such as "Rp". */
  prefix?: string;
  /** Called with each new value, besides updating the field. */
  onValueChange?: (value: string) => void;
};

/** A labelled paint-white input with its validation error or hint. */
export function TextField({
  field,
  label,
  hint,
  optional,
  prefix,
  onValueChange,
  className,
  ...input
}: TextFieldProps) {
  const error = fieldError(field);
  const messageId = `${field.name}-message`;

  return (
    <div className="flex flex-col gap-1.5">
      <FieldLabel htmlFor={field.name} optional={optional}>
        {label}
      </FieldLabel>
      <div className="relative">
        {prefix ? (
          <span
            aria-hidden="true"
            className="text-ink-soft pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-sm font-bold"
          >
            {prefix}
          </span>
        ) : null}
        <input
          id={field.name}
          name={field.name}
          value={field.state.value}
          onChange={(event) => {
            field.handleChange(event.target.value);
            onValueChange?.(event.target.value);
          }}
          onBlur={field.handleBlur}
          aria-invalid={error ? true : undefined}
          aria-describedby={error || hint ? messageId : undefined}
          className={cn(
            controlClass(Boolean(error)),
            "h-11",
            prefix && "pl-10",
            className,
          )}
          {...input}
        />
      </div>
      <FieldMessage id={messageId} error={error} hint={hint} />
    </div>
  );
}

type TextAreaFieldProps = Omit<
  TextareaHTMLAttributes<HTMLTextAreaElement>,
  "id" | "name" | "value" | "onChange" | "onBlur"
> & {
  field: AnyFieldApi;
  label: string;
  hint?: string;
  optional?: boolean;
};

/** A labelled multi-line field for notes and descriptions. */
export function TextAreaField({
  field,
  label,
  hint,
  optional,
  className,
  ...input
}: TextAreaFieldProps) {
  const error = fieldError(field);
  const messageId = `${field.name}-message`;

  return (
    <div className="flex flex-col gap-1.5">
      <FieldLabel htmlFor={field.name} optional={optional}>
        {label}
      </FieldLabel>
      <textarea
        id={field.name}
        name={field.name}
        value={field.state.value}
        onChange={(event) => field.handleChange(event.target.value)}
        onBlur={field.handleBlur}
        aria-invalid={error ? true : undefined}
        aria-describedby={error || hint ? messageId : undefined}
        rows={3}
        className={cn(
          controlClass(Boolean(error)),
          "min-h-24 py-2 text-pretty",
          className,
        )}
        {...input}
      />
      <FieldMessage id={messageId} error={error} hint={hint} />
    </div>
  );
}

export type SelectOption = { value: string; label: string };

type SelectControlProps = Omit<
  SelectHTMLAttributes<HTMLSelectElement>,
  "value" | "onChange"
> & {
  value: string;
  onValueChange: (value: string) => void;
  /** Label of the empty option; it is also what a cleared value reads as. */
  placeholder: string;
  options: SelectOption[];
  invalid?: boolean;
};

/** A native select (the best picker on a phone) drawn as a paint-white field. */
export function SelectControl({
  value,
  onValueChange,
  placeholder,
  options,
  invalid = false,
  className,
  ...select
}: SelectControlProps) {
  return (
    <div className="relative">
      <select
        value={value}
        onChange={(event) => onValueChange(event.target.value)}
        aria-invalid={invalid ? true : undefined}
        className={cn(
          controlClass(invalid),
          "h-11 appearance-none pr-10",
          value === "" && "text-ink-soft",
          className,
        )}
        {...select}
      >
        <option value="">{placeholder}</option>
        {options.map((option) => (
          <option key={option.value} value={option.value} className="text-ink">
            {option.label}
          </option>
        ))}
      </select>
      <ChevronDown
        aria-hidden="true"
        className="text-ink pointer-events-none absolute top-1/2 right-3 size-5 -translate-y-1/2"
        strokeWidth={2.5}
      />
    </div>
  );
}

type SelectFieldProps = {
  field: AnyFieldApi;
  label: string;
  placeholder: string;
  options: SelectOption[];
  hint?: string;
  optional?: boolean;
  disabled?: boolean;
  onValueChange?: (value: string) => void;
};

/** A labelled native select bound to a TanStack Form field. */
export function SelectField({
  field,
  label,
  placeholder,
  options,
  hint,
  optional,
  disabled,
  onValueChange,
}: SelectFieldProps) {
  const error = fieldError(field);
  const messageId = `${field.name}-message`;

  return (
    <div className="flex flex-col gap-1.5">
      <FieldLabel htmlFor={field.name} optional={optional}>
        {label}
      </FieldLabel>
      <SelectControl
        id={field.name}
        name={field.name}
        value={field.state.value}
        onValueChange={(value) => {
          field.handleChange(value);
          onValueChange?.(value);
        }}
        onBlur={field.handleBlur}
        placeholder={placeholder}
        options={options}
        invalid={Boolean(error)}
        disabled={disabled}
        aria-describedby={error || hint ? messageId : undefined}
      />
      <FieldMessage id={messageId} error={error} hint={hint} />
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

/**
 * The full-width yellow sign plate that submits a form. `form` targets a form
 * elsewhere in the page, such as one in a scrolling body above a sticky footer.
 */
export function SubmitPlate({
  pending,
  form,
  children,
}: {
  pending: boolean;
  form?: string;
  children: ReactNode;
}) {
  return (
    <button
      type="submit"
      form={form}
      disabled={pending}
      className={cn(signButtonClass, "disabled:bg-sign-soft mt-1 h-11 w-full")}
    >
      {pending ? "Memproses…" : children}
    </button>
  );
}
