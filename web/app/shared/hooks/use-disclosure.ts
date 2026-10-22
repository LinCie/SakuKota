import { useState } from "react";

/** Open state for a drawer or dialog. */
export function useDisclosure() {
  const [open, setOpen] = useState(false);
  return { open, setOpen };
}
