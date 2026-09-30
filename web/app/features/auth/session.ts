import { useSyncExternalStore } from "react";

import type { LoginResponse } from "./types";

/** The signed-in user and their token pair, persisted across reloads. */
export type Session = LoginResponse;

// The backend returns tokens in the response body (no cookies), so the
// session lives in localStorage. Components re-render through `useSession`
// whenever it changes.
const storageKey = "rantau:session";
const listeners = new Set<() => void>();
let session = readSession();

function readSession(): Session | null {
  try {
    return JSON.parse(localStorage.getItem(storageKey) ?? "null");
  } catch {
    return null;
  }
}

function writeSession(next: Session | null) {
  session = next;
  if (next) localStorage.setItem(storageKey, JSON.stringify(next));
  else localStorage.removeItem(storageKey);
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getSession() {
  return session;
}

export function saveSession(next: Session) {
  writeSession(next);
}

export function clearSession() {
  writeSession(null);
}

export function useSession() {
  return useSyncExternalStore(subscribe, getSession);
}
