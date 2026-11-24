import { cn } from "~/shared/cn";

type UserAvatarProps = {
  name: string;
  /** The profile photo; without one, the name's initial shows instead. */
  url: string | null;
  /** Empty when the name is shown next to the avatar. */
  alt?: string;
  /** Size classes, e.g. `size-20 text-2xl`. */
  className?: string;
};

/** A user's round profile photo, or their initial when they have none. */
export function UserAvatar({
  name,
  url,
  alt = "",
  className,
}: UserAvatarProps) {
  if (url) {
    return (
      <img
        src={url}
        alt={alt}
        className={cn("size-8 shrink-0 rounded-full object-cover", className)}
      />
    );
  }
  return (
    <span
      role={alt ? "img" : undefined}
      aria-label={alt || undefined}
      aria-hidden={alt ? undefined : true}
      className={cn(
        "bg-muted text-muted-foreground grid size-8 shrink-0 place-items-center rounded-full text-sm font-semibold",
        className,
      )}
    >
      {name.trim().charAt(0).toUpperCase()}
    </span>
  );
}
