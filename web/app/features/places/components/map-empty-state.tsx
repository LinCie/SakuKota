import { cn } from "~/shared/cn";
import { signButtonClass } from "~/shared/components/menu-board";

export type EmptyState = {
  message: string;
  /** The button that offers a way out, such as clearing the search. */
  action: string;
  onAction: () => void;
};

/** The board laid over the map when no places match. */
export function MapEmptyState({ message, action, onAction }: EmptyState) {
  return (
    <div className="bg-paint/70 absolute inset-0 z-1000 grid place-items-center p-4">
      <div className="border-ink bg-paint flex max-w-sm flex-col items-center gap-3 rounded-[5px] border-2 p-5 text-center shadow-[0_8px_24px_rgb(26_26_23/0.2)]">
        <p className="text-ink font-semibold text-pretty">{message}</p>
        <button
          type="button"
          onClick={onAction}
          className={cn(signButtonClass, "h-10")}
        >
          {action}
        </button>
      </div>
    </div>
  );
}
