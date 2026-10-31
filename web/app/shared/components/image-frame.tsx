import { useState } from "react";

import { cn } from "~/shared/cn";

import { MissingSlot } from "./missing-slot";

type ImageFrameProps = {
  src: string | null;
  alt: string;
  /** Names what is missing when there is no image or it fails to load. */
  missing: string;
  /** Aspect ratio class of the frame, e.g. `aspect-8/5`. */
  aspect: string;
};

/** An image in a bordered frame, or a drawn slot when it is missing or fails to load. */
export function ImageFrame({ src, alt, missing, aspect }: ImageFrameProps) {
  // Remembers which image failed, so a new src gets its own chance to load.
  const [failedSrc, setFailedSrc] = useState<string | null>(null);

  if (!src || failedSrc === src) {
    return (
      <div
        className={cn(
          "border-ink/30 grid w-full place-items-center rounded-[3px] border-2 border-dashed",
          aspect,
        )}
      >
        <MissingSlot>{missing}</MissingSlot>
      </div>
    );
  }

  return (
    <img
      src={src}
      alt={alt}
      loading="lazy"
      onError={() => setFailedSrc(src)}
      className={cn(
        "border-ink bg-paint-deep w-full rounded-[3px] border-2 object-cover",
        aspect,
      )}
    />
  );
}
