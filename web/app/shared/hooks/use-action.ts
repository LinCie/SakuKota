import { useState } from "react";

/**
 * Runs an async task with pending and error state. The task resolves to an
 * error message, or null on success; `run` resolves to whether it succeeded.
 */
export function useAction() {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function run(task: () => Promise<string | null>): Promise<boolean> {
    setPending(true);
    setError(null);
    try {
      const message = await task();
      if (message) {
        setError(message);
        return false;
      }
      return true;
    } catch {
      setError("Tidak bisa terhubung ke server. Coba lagi.");
      return false;
    } finally {
      setPending(false);
    }
  }

  return { pending, error, run, clearError: () => setError(null) };
}
