import { useQueryErrorResetBoundary } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useEffect } from "react";

import { ErrorState } from "~/shared/components/error-state";

/** The error shown in place of an admin page whose data failed to load. */
export function AdminRouteError() {
  const router = useRouter();
  const queryErrorResetBoundary = useQueryErrorResetBoundary();

  // Lets failed queries fetch again once the page re-renders.
  useEffect(() => {
    queryErrorResetBoundary.reset();
  }, [queryErrorResetBoundary]);

  return (
    <ErrorState
      title="Halaman ini belum bisa dimuat."
      message="Periksa koneksimu, lalu coba lagi. Jika terus terjadi, hubungi tim pengembang."
      onRetry={() => void router.invalidate()}
    />
  );
}
