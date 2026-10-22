import { Button } from "./button";

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
      className="border-chili/40 bg-paint flex flex-col items-start gap-3 rounded-[5px] border p-5"
    >
      <p className="font-bold">{title}</p>
      {message ? (
        <p className="text-ink-soft text-sm text-pretty">{message}</p>
      ) : null}
      <Button variant="secondary" onClick={onRetry}>
        Coba lagi
      </Button>
    </div>
  );
}
