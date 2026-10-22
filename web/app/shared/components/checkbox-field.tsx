import { Checkbox } from "@base-ui/react/checkbox";
import { Check } from "lucide-react";

/** A labelled checkbox for yes-or-no settings in a form. */
export function CheckboxField({
  id,
  label,
  checked,
  onCheckedChange,
  hint,
}: {
  id: string;
  label: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  hint?: string;
}) {
  return (
    <div className="flex items-start gap-3">
      <Checkbox.Root
        id={id}
        checked={checked}
        onCheckedChange={onCheckedChange}
        className="border-ink/40 bg-paint focus-visible:outline-cat data-checked:border-board data-checked:bg-board data-checked:text-paint mt-0.5 grid size-5 shrink-0 place-items-center rounded-[3px] border transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
      >
        <Checkbox.Indicator>
          <Check aria-hidden="true" className="size-3.5" strokeWidth={3} />
        </Checkbox.Indicator>
      </Checkbox.Root>
      <div className="flex flex-col gap-0.5">
        <label htmlFor={id} className="text-sm font-semibold">
          {label}
        </label>
        {hint ? <p className="text-ink-soft text-sm">{hint}</p> : null}
      </div>
    </div>
  );
}
