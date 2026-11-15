import { Search, X } from "lucide-react";
import { useId } from "react";

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
  // The field renders in the header from 640px and in the paint-white category
  // strip below it, so its focus border is yellow from 640px and blue below.
  const id = useId();

  return (
    <form
      role="search"
      onSubmit={(event) => {
        event.preventDefault();
        onSearch();
      }}
      className={cn("relative", className)}
    >
      <label htmlFor={id} className="sr-only">
        Cari tempat
      </label>
      <Search
        aria-hidden="true"
        className="text-ink-soft pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2"
        strokeWidth={2.5}
      />
      <input
        id={id}
        type="search"
        value={draft}
        onChange={(event) => onDraftChange(event.target.value)}
        placeholder="Cari warung, apotek, nama tempat…"
        autoComplete="off"
        className="border-ink bg-paint text-ink placeholder:text-ink-soft focus-visible:border-cat sm:focus-visible:border-sign h-11 w-full rounded-[3px] border-2 pr-11 pl-10 text-ellipsis focus-visible:outline-none [&::-webkit-search-cancel-button]:appearance-none"
      />
      {draft ? (
        <button
          type="button"
          onClick={onClearSearch}
          aria-label="Hapus pencarian"
          className="text-ink hover:bg-paint-deep focus-visible:bg-paint-deep focus-visible:outline-cat absolute top-1/2 right-1.5 grid size-8 -translate-y-1/2 place-items-center rounded-[3px] transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2"
        >
          <X aria-hidden="true" className="size-4" strokeWidth={2.5} />
        </button>
      ) : null}
    </form>
  );
}
