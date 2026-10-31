import { Badge } from "~/shared/components/badge";

import type { SuggestionStatus } from "../types";

export const STATUS_LABELS: Record<SuggestionStatus, string> = {
  pending: "Menunggu",
  approved: "Disetujui",
  rejected: "Ditolak",
};

// State by mark: a pending suggestion is still open, so it is a dashed
// outline; a reviewed one is a solid plate.
const STATUS_CLASSES: Record<SuggestionStatus, string> = {
  pending: "border-ink/60 border-dashed text-ink",
  approved: "border-board bg-board text-paint",
  rejected: "border-chili bg-chili text-paint",
};

export function SuggestionStatusBadge({
  status,
}: {
  status: SuggestionStatus;
}) {
  return (
    <Badge className={STATUS_CLASSES[status]}>{STATUS_LABELS[status]}</Badge>
  );
}
