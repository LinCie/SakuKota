import type { ReactFormExtendedApi } from "@tanstack/react-form";

/**
 * A TanStack form for `TValues`, typed loosely so field components can take
 * the form their drawer created, whatever validators it was given.
 */
/* eslint-disable @typescript-eslint/no-explicit-any */
export type AppFormApi<TValues> = ReactFormExtendedApi<
  TValues,
  any,
  any,
  any,
  any,
  any,
  any,
  any,
  any,
  any,
  any,
  any
>;
/* eslint-enable @typescript-eslint/no-explicit-any */
