import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";

import { cn } from "~/shared/cn";
import {
  FieldLabel,
  FieldMessage,
  SelectControl,
} from "~/shared/components/field";
import { buttonVariants } from "~/shared/components/ui/button";
import { Input } from "~/shared/components/ui/input";
import { rangeOf, textOf, type AttributeValue } from "../values";
import type { CategoryField } from "../types";

type CategoryFieldInputProps = {
  field: CategoryField;
  value: AttributeValue | undefined;
  onChange: (value: AttributeValue) => void;
  error?: string;
};

// The picked choice fills pandan; the others are outline pills.
const choiceClass = cn(
  buttonVariants({ variant: "outline" }),
  "data-checked:border-primary data-checked:bg-primary data-checked:text-primary-foreground data-checked:hover:bg-primary/90 data-checked:hover:text-primary-foreground min-w-16",
);

// A real button: a drawer's swipe handler captures the pointer from anything
// else, so a click on a `span role="radio"` never reaches it.
const radioProps = {
  nativeButton: true,
  render: <button type="button" />,
  className: choiceClass,
};

/**
 * One of a category's custom fields, drawn by its data type. Required fields
 * carry no mark; optional ones say "opsional".
 */
export function CategoryFieldInput({
  field,
  value,
  onChange,
  error,
}: CategoryFieldInputProps) {
  const id = `attr-${field.key}`;
  const messageId = `${id}-message`;
  const optional = !field.required;
  const described = error ? messageId : undefined;
  const invalid = Boolean(error);

  let control;
  switch (field.data_type) {
    case "boolean":
      control = (
        <RadioGroup
          value={textOf(value)}
          onValueChange={(next) => onChange(String(next))}
          aria-labelledby={`${id}-label`}
          aria-describedby={described}
          className="flex flex-wrap gap-2"
        >
          <Radio.Root value="true" {...radioProps}>
            Ya
          </Radio.Root>
          <Radio.Root value="false" {...radioProps}>
            Tidak
          </Radio.Root>
          {optional ? (
            <Radio.Root value="" {...radioProps}>
              Belum tahu
            </Radio.Root>
          ) : null}
        </RadioGroup>
      );
      break;
    case "select":
      control = (
        <SelectControl
          id={id}
          value={textOf(value)}
          onValueChange={onChange}
          placeholder="Pilih salah satu"
          options={(field.options ?? []).map((option) => ({
            value: option,
            label: option,
          }))}
          invalid={invalid}
          aria-describedby={described}
        />
      );
      break;
    case "range": {
      const range = rangeOf(value);
      control = (
        <div
          role="group"
          aria-labelledby={`${id}-label`}
          aria-describedby={described}
          className="grid grid-cols-2 gap-3"
        >
          <Input
            type="number"
            step="any"
            inputMode="decimal"
            aria-label={`${field.label}, dari`}
            placeholder="Dari"
            value={range.min}
            onChange={(event) =>
              onChange({ ...range, min: event.target.value })
            }
            aria-invalid={invalid ? true : undefined}
          />
          <Input
            type="number"
            step="any"
            inputMode="decimal"
            aria-label={`${field.label}, sampai`}
            placeholder="Sampai"
            value={range.max}
            onChange={(event) =>
              onChange({ ...range, max: event.target.value })
            }
            aria-invalid={invalid ? true : undefined}
          />
        </div>
      );
      break;
    }
    case "number":
      control = (
        <Input
          id={id}
          type="number"
          step="any"
          inputMode="decimal"
          value={textOf(value)}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={invalid ? true : undefined}
          aria-describedby={described}
        />
      );
      break;
    default:
      // Text, and any type this frontend doesn't know yet.
      control = (
        <Input
          id={id}
          type={field.data_type === "date" ? "date" : "text"}
          value={textOf(value)}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={invalid ? true : undefined}
          aria-describedby={described}
        />
      );
  }

  return (
    <div className="flex flex-col gap-1.5">
      <span id={`${id}-label`}>
        <FieldLabel
          htmlFor={
            field.data_type === "boolean" || field.data_type === "range"
              ? undefined
              : id
          }
          optional={optional}
        >
          {field.label}
        </FieldLabel>
      </span>
      {control}
      <FieldMessage id={messageId} error={error} />
    </div>
  );
}
