import { ChevronLeft, ChevronRight } from "lucide-react";

import { formatNumber } from "~/shared/format";

import { Button } from "./ui/button";

type PaginationProps = {
  /** The current page, counting from 1. */
  page: number;
  pageCount: number;
  onPageChange: (page: number) => void;
};

/** Previous and next buttons for a paged table, with the current page. */
export function Pagination({ page, pageCount, onPageChange }: PaginationProps) {
  return (
    <nav
      aria-label="Halaman"
      className="flex items-center justify-end gap-3 text-sm"
    >
      <p className="text-muted-foreground tabular-nums">
        Halaman {formatNumber(page)} dari {formatNumber(pageCount)}
      </p>
      <Button
        variant="outline"
        size="icon-sm"
        aria-label="Halaman sebelumnya"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
      >
        <ChevronLeft aria-hidden="true" />
      </Button>
      <Button
        variant="outline"
        size="icon-sm"
        aria-label="Halaman berikutnya"
        disabled={page >= pageCount}
        onClick={() => onPageChange(page + 1)}
      >
        <ChevronRight aria-hidden="true" />
      </Button>
    </nav>
  );
}
