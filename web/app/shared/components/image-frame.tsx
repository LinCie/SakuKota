import { ImageOff } from "lucide-react";
import { useState } from "react";

import { cn } from "~/shared/cn";

type ImageFrameProps = {
  src: string | null;
  alt: string;
  /** Names what is missing when there is no image or it fails to load. */
  missing: string;
  /** Aspect ratio class of the frame, e.g. `aspect-8/5`. */
  aspect: string;
  /** Content type of the file; videos are played with controls. */
  contentType?: string;
};

/** An image or video in a rounded frame, or a muted frame naming what is missing when there is none or it fails to load. */
export function ImageFrame({
  src,
  alt,
  missing,
  aspect,
  contentType,
}: ImageFrameProps) {
  // Remembers which image failed, so a new src gets its own chance to load.
  const [failedSrc, setFailedSrc] = useState<string | null>(null);

  if (!src || failedSrc === src) {
    return (
      <div
        className={cn(
          "bg-muted text-muted-foreground flex w-full flex-col items-center justify-center gap-2 rounded-xl p-3 text-center text-sm",
          aspect,
        )}
      >
        <ImageOff aria-hidden="true" className="size-6" />
        {missing}
      </div>
    );
  }

  if (contentType?.startsWith("video/")) {
    return (
      <video
        src={src}
        aria-label={alt}
        controls
        preload="metadata"
        onError={() => setFailedSrc(src)}
        className={cn("bg-muted w-full rounded-xl object-cover", aspect)}
      />
    );
  }

  return (
    <img
      src={src}
      alt={alt}
      loading="lazy"
      onError={() => setFailedSrc(src)}
      className={cn("bg-muted w-full rounded-xl object-cover", aspect)}
    />
  );
}
