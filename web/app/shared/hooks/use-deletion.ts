import { useState } from "react";

import { useAction } from "./use-action";

/**
 * Asks before deleting: holds the item awaiting confirmation, runs `remove`
 * on confirm, and clears the confirmation once it succeeds. `remove` resolves
 * to an error message, or null on success.
 */
export function useDeletion<T>(
  remove: (item: T) => Promise<string | null>,
  onRemoved: (item: T) => void,
) {
  const [target, setTarget] = useState<T | null>(null);
  const action = useAction();

  function close() {
    setTarget(null);
    action.clearError();
  }

  async function confirm() {
    if (target === null) return;
    const removed = await action.run(() => remove(target));
    if (removed) {
      onRemoved(target);
      close();
    }
  }

  return {
    target,
    pending: action.pending,
    error: action.error,
    request: setTarget,
    close,
    confirm,
  };
}
