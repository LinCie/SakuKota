import { Badge } from "~/shared/components/ui/badge";

import type { SuggestionStatus } from "../types";

export const STATUS_LABELS: Record<SuggestionStatus, string> = {
  pending: "Menunggu",
  approved: "Disetujui",
  rejected: "Ditolak",
};

// A pending suggestion is still open, so it stays quiet; a reviewed one
// takes its outcome's tint.
const STATUS_CLASSES: Record<SuggestionStatus, string> = {
  pending: "bg-muted text-muted-foreground",
  approved: "bg-pandan-soft text-pandan-deep",
  rejected: "bg-sambal-soft text-sambal-deep",
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
