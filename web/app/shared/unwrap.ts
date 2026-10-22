import type { ApiResult } from "./api";

/** The data of a successful result; throws the server's message otherwise, for route loaders. */
export function unwrap<T, C extends string>(result: ApiResult<T, C>): T {
  if (!result.ok) throw new Error(result.error.message);
  return result.data;
}
