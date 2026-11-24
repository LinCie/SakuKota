import { Button } from "./ui/button";

/** A page or panel that failed to load, with a retry. */
export function ErrorState({
  title,
  message,
  onRetry,
}: {
  title: string;
  message?: string;
  onRetry: () => void;
}) {
  return (
    <div
      role="alert"
      className="bg-card ring-foreground/10 flex flex-col items-start gap-3 rounded-xl p-5 ring-1"
    >
      <p className="font-semibold">{title}</p>
      {message ? (
        <p className="text-muted-foreground text-sm text-pretty">{message}</p>
      ) : null}
      <Button variant="outline" size="sm" onClick={onRetry}>
        Coba lagi
      </Button>
    </div>
  );
}
