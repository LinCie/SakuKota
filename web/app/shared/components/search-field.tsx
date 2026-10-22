import { Search } from "lucide-react";

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
  return (
    <label className="relative block w-full max-w-sm">
      <span className="sr-only">{label}</span>
      <Search
        aria-hidden="true"
        className="text-ink-soft pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
        strokeWidth={2.5}
      />
      <input
        type="search"
        value={value}
        onChange={(event) => onValueChange(event.target.value)}
        placeholder={placeholder}
        autoComplete="off"
        className="border-ink/20 bg-paint text-ink placeholder:text-ink-soft focus-visible:outline-cat h-9 w-full rounded-[3px] border pr-3 pl-9 text-sm focus-visible:outline-2 focus-visible:outline-offset-2"
      />
    </label>
  );
}
