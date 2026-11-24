import { Search, X } from "lucide-react";
import { useId } from "react";

import { Button } from "~/shared/components/ui/button";

type SearchFieldProps = {
  /** The search text as typed, before it is debounced into the URL. */
  draft: string;
  onDraftChange: (draft: string) => void;
  onSearch: () => void;
  onClearSearch: () => void;
};

/**
 * The place search, a borderless segment of the header's search bar, with a
 * clear button once it holds text. The bar draws the frame and focus ring.
 */
export function SearchField({
  draft,
  onDraftChange,
  onSearch,
  onClearSearch,
}: SearchFieldProps) {
  const id = useId();

  return (
    <form
      role="search"
      onSubmit={(event) => {
        event.preventDefault();
        onSearch();
      }}
      className="flex h-full min-w-0 flex-1 items-center"
    >
      <label
        htmlFor={id}
        className="text-muted-foreground flex h-full shrink-0 cursor-text items-center pr-2 pl-3.5 sm:pl-3"
      >
        <Search aria-hidden="true" className="size-5" />
        <span className="sr-only">Cari tempat</span>
      </label>
      <input
        id={id}
        type="search"
        value={draft}
        onChange={(event) => onDraftChange(event.target.value)}
        placeholder="Cari warung, apotek, nama tempat…"
        autoComplete="off"
        className="placeholder:text-muted-foreground h-full min-w-0 flex-1 bg-transparent pr-3 text-base text-ellipsis outline-none md:text-sm [&::-webkit-search-cancel-button]:appearance-none"
      />
      {draft ? (
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={onClearSearch}
          aria-label="Hapus pencarian"
          className="text-muted-foreground mr-1 -ml-1"
        >
          <X aria-hidden="true" />
        </Button>
      ) : null}
    </form>
  );
}
