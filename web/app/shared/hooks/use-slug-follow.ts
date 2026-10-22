import { useRef } from "react";

import { slugify } from "~/shared/slug";

/**
 * Keeps a slug in step with its name until the slug is edited by hand. Keep
 * the returned object for the life of one form; it resets when the form unmounts.
 */
export function useSlugFollow(followSlug: boolean) {
  const edited = useRef(!followSlug);

  return {
    /** The slug for a name, or null once the slug was edited by hand. */
    slugFor: (name: string): string | null =>
      edited.current ? null : slugify(name),
    /** Records a hand edit, which stops the slug following the name. */
    markEdited: () => {
      edited.current = true;
    },
  };
}
