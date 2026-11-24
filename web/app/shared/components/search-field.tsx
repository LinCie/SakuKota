import { Search } from "lucide-react";
import { useId } from "react";

import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "~/shared/components/ui/input-group";

/** A search box for filtering the rows of a table. */
export function SearchField({
  value,
  onValueChange,
  label,
  placeholder,
}: {
  value: string;
  onValueChange: (value: string) => void;
  label: string;
  placeholder: string;
}) {
  const id = useId();

  return (
    <div className="w-full max-w-sm">
      <label htmlFor={id} className="sr-only">
        {label}
      </label>
      <InputGroup className="h-9 md:h-9">
        <InputGroupAddon>
          <Search aria-hidden="true" />
        </InputGroupAddon>
        <InputGroupInput
          id={id}
          type="search"
          value={value}
          onChange={(event) => onValueChange(event.target.value)}
          placeholder={placeholder}
          autoComplete="off"
        />
      </InputGroup>
    </div>
  );
}
