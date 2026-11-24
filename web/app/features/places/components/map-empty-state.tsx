import { Button } from "~/shared/components/ui/button";

export type EmptyState = {
  message: string;
  /** The button that offers a way out, such as clearing the search. */
  action: string;
  onAction: () => void;
};

/** The card laid over the map when no places match. */
export function MapEmptyState({ message, action, onAction }: EmptyState) {
  return (
    <div className="bg-background/70 absolute inset-0 z-1000 grid place-items-center p-4">
      <div className="bg-card shadow-float flex max-w-sm flex-col items-center gap-4 rounded-xl p-5 text-center">
        <p className="text-foreground font-medium text-pretty">{message}</p>
        <Button onClick={onAction}>{action}</Button>
      </div>
    </div>
  );
}
