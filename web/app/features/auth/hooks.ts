import { useSyncExternalStore } from "react";

import { getSession, subscribeSession } from "./session";

/** The current session; re-renders whenever it changes. */
export function useSession() {
  return useSyncExternalStore(subscribeSession, getSession);
}
