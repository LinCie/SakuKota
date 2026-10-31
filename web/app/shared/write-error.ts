import { ApiError } from "./api";

/** A form-level message for a failed write. */
export function writeErrorMessage(error: Error): string {
  if (!(error instanceof ApiError)) {
    return "Tidak bisa terhubung ke server. Coba lagi.";
  }
  if (error.status === 401) return "Sesimu berakhir. Masuk lagi, lalu ulangi.";
  if (error.status === 403) return "Akun ini tidak punya akses admin.";
  if (error.code === "INTERNAL_ERROR") {
    return "Terjadi kesalahan di server. Coba lagi.";
  }
  return `Server menolak isian ini: ${error.message}.`;
}

/** Whether `error` is a backend error with the given code. */
export function hasErrorCode(error: Error, code: string): boolean {
  return error instanceof ApiError && error.code === code;
}
