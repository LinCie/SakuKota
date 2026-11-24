import { Search, X } from "lucide-react";
import { useId } from "react";

import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "~/shared/components/ui/input-group";
import { cn } from "~/shared/cn";

type SearchFieldProps = {
  /** The search text as typed, before it is debounced into the URL. */
  draft: string;
  onDraftChange: (draft: string) => void;
  onSearch: () => void;
  onClearSearch: () => void;
  className?: string;
};

/** The place search field, with a clear button once it holds text. */
export function SearchField({
  draft,
  onDraftChange,
  onSearch,
  onClearSearch,
  className,
}: SearchFieldProps) {
  const id = useId();

  return (
    <form
      role="search"
      onSubmit={(event) => {
        event.preventDefault();
        onSearch();
      }}
      className={className}
    >
      <label htmlFor={id} className="sr-only">
        Cari tempat
      </label>
      <InputGroup>
        <InputGroupAddon>
          <Search aria-hidden="true" className="size-5" />
        </InputGroupAddon>
        <InputGroupInput
          id={id}
          type="search"
          value={draft}
          onChange={(event) => onDraftChange(event.target.value)}
          placeholder="Cari warung, apotek, nama tempat…"
          autoComplete="off"
          className={cn(
            "text-ellipsis [&::-webkit-search-cancel-button]:appearance-none",
            draft ? null : "pr-4",
          )}
        />
        {draft ? (
          <InputGroupAddon align="inline-end">
            <InputGroupButton
              size="icon-sm"
              onClick={onClearSearch}
              aria-label="Hapus pencarian"
            >
              <X aria-hidden="true" />
            </InputGroupButton>
          </InputGroupAddon>
        ) : null}
      </InputGroup>
    </form>
  );
}
