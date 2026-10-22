import type { ErrorComponentProps } from "@tanstack/react-router";

import { ErrorState } from "~/shared/components/error-state";

/** The error shown in place of an admin page whose data failed to load. */
export function AdminRouteError({ reset }: ErrorComponentProps) {
  return (
    <ErrorState
      title="Halaman ini belum bisa dimuat."
      message="Periksa koneksimu, lalu coba lagi. Jika terus terjadi, hubungi tim pengembang."
      onRetry={reset}
    />
  );
}
