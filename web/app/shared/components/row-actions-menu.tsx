import { Menu } from "@base-ui/react/menu";
import { MoreHorizontal, type LucideIcon } from "lucide-react";

import { cn } from "~/shared/cn";

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
    <Menu.Root>
      <Menu.Trigger
        aria-label={label}
        className="text-ink-soft hover:bg-paint-deep hover:text-ink focus-visible:outline-cat grid size-8 place-items-center rounded-[3px] transition-colors focus-visible:outline-2 focus-visible:outline-offset-2"
      >
        <MoreHorizontal
          aria-hidden="true"
          className="size-4"
          strokeWidth={2.5}
        />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner sideOffset={4} align="end" className="z-50">
          <Menu.Popup className="border-ink/15 bg-paint text-ink min-w-40 rounded-[5px] border p-1 shadow-[0_8px_24px_rgb(26_26_23/0.2)] outline-none">
            {actions.map((action) => {
              const Icon = action.icon;
              return (
                <Menu.Item
                  key={action.label}
                  onClick={action.onSelect}
                  className={cn(
                    "data-highlighted:bg-paint-deep flex h-9 cursor-pointer items-center gap-2.5 rounded-[3px] px-2.5 text-sm font-medium outline-none",
                    action.destructive && "text-chili",
                  )}
                >
                  <Icon
                    aria-hidden="true"
                    className="size-4"
                    strokeWidth={2.5}
                  />
                  {action.label}
                </Menu.Item>
              );
            })}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}
