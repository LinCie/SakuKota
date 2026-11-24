import { MoreHorizontal, type LucideIcon } from "lucide-react";

import { Button } from "~/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "~/shared/components/ui/dropdown-menu";

export type RowAction = {
  label: string;
  icon: LucideIcon;
  onSelect: () => void;
  destructive?: boolean;
};

/** A row's "more" button opening a menu of actions. */
export function RowActionsMenu({
  label,
  actions,
}: {
  /** Names the row for assistive technology, e.g. "Aksi untuk Warung Pak Eko". */
  label: string;
  actions: RowAction[];
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={label}
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground"
          />
        }
      >
        <MoreHorizontal aria-hidden="true" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-auto min-w-44">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <DropdownMenuItem
              key={action.label}
              onClick={action.onSelect}
              variant={action.destructive ? "destructive" : "default"}
              className="cursor-pointer"
            >
              <Icon aria-hidden="true" />
              {action.label}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
