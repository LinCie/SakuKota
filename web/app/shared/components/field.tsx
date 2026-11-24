import type { AnyFieldApi } from "@tanstack/react-form";
import type {
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from "react";

import { cn } from "~/shared/cn";

import { Input } from "./ui/input";
import { NativeSelect, NativeSelectOption } from "./ui/native-select";
import { Textarea } from "./ui/textarea";

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
    <label htmlFor={htmlFor} className="text-foreground text-sm font-medium">
      {children}
      {optional ? (
        <span className="text-muted-foreground ml-1.5 font-normal">
          opsional
        </span>
      ) : null}
    </label>
  );
}

/** The error (clay red) or hint (muted) line under a field. */
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
      <p id={id} className="text-destructive text-sm">
        {error}
      </p>
    );
  }
  return hint ? (
    <p id={id} className="text-muted-foreground text-sm">
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

/** A labelled input with its validation error or hint. */
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
            className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-sm font-medium"
          >
            {prefix}
          </span>
        ) : null}
        <Input
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
          className={cn(prefix && "pl-10", className)}
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
      <Textarea
        id={field.name}
        name={field.name}
        value={field.state.value}
        onChange={(event) => field.handleChange(event.target.value)}
        onBlur={field.handleBlur}
        aria-invalid={error ? true : undefined}
        aria-describedby={error || hint ? messageId : undefined}
        rows={3}
        className={cn("min-h-24 text-pretty", className)}
        {...input}
      />
      <FieldMessage id={messageId} error={error} hint={hint} />
    </div>
  );
}

export type SelectOption = { value: string; label: string };

type SelectControlProps = Omit<
  SelectHTMLAttributes<HTMLSelectElement>,
  "value" | "onChange" | "size"
> & {
  /** `sm` is the 36px height of admin toolbars. */
  size?: "sm" | "default";
  value: string;
  onValueChange: (value: string) => void;
  /** Label of the empty option; it is also what a cleared value reads as. */
  placeholder: string;
  options: SelectOption[];
  invalid?: boolean;
};

/** A native select, the best picker on a phone, drawn as a pill field. */
export function SelectControl({
  value,
  onValueChange,
  placeholder,
  options,
  invalid = false,
  size,
  className,
  ...select
}: SelectControlProps) {
  return (
    <NativeSelect
      value={value}
      onChange={(event) => onValueChange(event.target.value)}
      aria-invalid={invalid ? true : undefined}
      size={size}
      className={cn(
        "w-full",
        value === "" ? "text-muted-foreground" : "text-foreground",
        className,
      )}
      {...select}
    >
      <NativeSelectOption value="">{placeholder}</NativeSelectOption>
      {options.map((option) => (
        <NativeSelectOption key={option.value} value={option.value}>
          {option.label}
        </NativeSelectOption>
      ))}
    </NativeSelect>
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
