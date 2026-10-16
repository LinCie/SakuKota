import type { ApiErrorResponse } from "./api";

/** A form-level message for a rejected admin write (create). */
export function adminWriteError(
  status: number,
  error: ApiErrorResponse,
): string {
  if (status === 401) return "Sesimu berakhir. Masuk lagi, lalu ulangi.";
  if (status === 403) return "Akun ini tidak punya akses admin.";
  if (error.code === "internal_error") {
    return "Terjadi kesalahan di server. Coba lagi.";
  }
  return `Server menolak isian ini: ${error.message}.`;
}
