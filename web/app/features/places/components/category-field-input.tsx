import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";

import { cn } from "~/shared/cn";
import {
  controlClass,
  FieldLabel,
  FieldMessage,
  SelectControl,
} from "~/shared/components/field";
import { rangeOf, textOf, type AttributeValue } from "../attribute-values";
import type { CategoryField } from "../types";

type CategoryFieldInputProps = {
  field: CategoryField;
  value: AttributeValue | undefined;
  onChange: (value: AttributeValue) => void;
  error?: string;
};

const choiceClass =
  "focus-visible:outline-cat inline-flex h-10 cursor-pointer items-center rounded-[3px] border-2 px-4 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 data-checked:border-ink data-checked:bg-ink data-checked:text-paint data-unchecked:border-ink/60 data-unchecked:hover:border-ink data-unchecked:border-dashed data-unchecked:hover:border-solid";

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
  const inputClass = cn(controlClass(invalid), "h-11");

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
          <input
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
            className={inputClass}
          />
          <input
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
            className={inputClass}
          />
        </div>
      );
      break;
    }
    default:
      control = (
        <input
          id={id}
          type={
            field.data_type === "number"
              ? "number"
              : field.data_type === "date"
                ? "date"
                : "text"
          }
          step={field.data_type === "number" ? "any" : undefined}
          inputMode={field.data_type === "number" ? "decimal" : undefined}
          value={textOf(value)}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={invalid ? true : undefined}
          aria-describedby={described}
          className={inputClass}
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
