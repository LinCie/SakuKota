import { ChevronLeft, ChevronRight } from "lucide-react";
import { useState } from "react";

import { cn } from "~/shared/cn";
import type { Media } from "~/shared/image-upload";

import { ImageFrame } from "./image-frame";

type MediaGalleryProps = {
  medias: Media[];
  alt: string;
  /** Names what is missing when there are no images. */
  missing: string;
  /** Aspect ratio class of the image, e.g. `aspect-8/5`. */
  aspect: string;
};

const arrowClass =
  "bg-paint/90 text-ink border-ink hover:bg-paint focus-visible:outline-cat absolute top-1/2 grid size-8 -translate-y-1/2 place-items-center rounded-[3px] border-2 focus-visible:outline-2";

// Full class names, so Tailwind finds them: one shift per image, up to maxMedias.
const shift = [
  "translate-x-0",
  "-translate-x-full",
  "-translate-x-[200%]",
  "-translate-x-[300%]",
  "-translate-x-[400%]",
  "-translate-x-[500%]",
  "-translate-x-[600%]",
  "-translate-x-[700%]",
  "-translate-x-[800%]",
  "-translate-x-[900%]",
];

/** A place's or sub-district's images, one at a time, sliding between them with arrows and dots. */
export function MediaGallery({
  medias,
  alt,
  missing,
  aspect,
}: MediaGalleryProps) {
  // The shown image is tracked by id, so it stays put when others are added or
  // removed, and falls back to the first when it is gone.
  const [currentId, setCurrentId] = useState<number | null>(null);

  if (medias.length === 0) {
    return (
      <ImageFrame src={null} alt={alt} missing={missing} aspect={aspect} />
    );
  }

  const found = medias.findIndex((media) => media.id === currentId);
  const index = found === -1 ? 0 : found;
  const media = medias[index];

  function show(next: number) {
    setCurrentId(medias[(next + medias.length) % medias.length].id);
  }

  if (medias.length === 1) {
    return (
      <ImageFrame
        src={media.url}
        contentType={media.content_type}
        alt={`${alt} ${media.position}`}
        missing={missing}
        aspect={aspect}
      />
    );
  }

  return (
    <div
      role="region"
      aria-roledescription="carousel"
      aria-label={alt}
      tabIndex={0}
      onKeyDown={(event) => {
        if (event.key === "ArrowLeft") {
          event.preventDefault();
          show(index - 1);
        } else if (event.key === "ArrowRight") {
          event.preventDefault();
          show(index + 1);
        }
      }}
      className="focus-visible:outline-cat flex flex-col gap-2 rounded-[3px] focus-visible:outline-2 focus-visible:outline-offset-2"
    >
      <div
        role="group"
        aria-roledescription="slide"
        aria-label={`${index + 1} dari ${medias.length}`}
        className="relative"
      >
        <div className="overflow-hidden rounded-[3px]">
          <ul
            className={cn(
              "flex transition-transform duration-300 ease-out motion-reduce:transition-none",
              shift[index],
            )}
          >
            {medias.map((m, i) => (
              <li
                key={m.id}
                aria-hidden={i !== index}
                className="w-full shrink-0"
              >
                <ImageFrame
                  src={m.url}
                  contentType={m.content_type}
                  alt={`${alt} ${m.position}`}
                  missing={missing}
                  aspect={aspect}
                />
              </li>
            ))}
          </ul>
        </div>
        <button
          type="button"
          aria-label="Gambar sebelumnya"
          onClick={() => show(index - 1)}
          className={cn(arrowClass, "left-2")}
        >
          <ChevronLeft
            aria-hidden="true"
            className="size-5"
            strokeWidth={2.5}
          />
        </button>
        <button
          type="button"
          aria-label="Gambar berikutnya"
          onClick={() => show(index + 1)}
          className={cn(arrowClass, "right-2")}
        >
          <ChevronRight
            aria-hidden="true"
            className="size-5"
            strokeWidth={2.5}
          />
        </button>
      </div>
      <div className="flex justify-center gap-1.5">
        {medias.map((m, i) => (
          <button
            key={m.id}
            type="button"
            aria-label={`Gambar ${i + 1}`}
            aria-current={i === index}
            onClick={() => show(i)}
            className={cn(
              "border-ink focus-visible:outline-cat size-2.5 rounded-full border focus-visible:outline-2",
              i === index ? "bg-ink" : "bg-paint",
            )}
          />
        ))}
      </div>
    </div>
  );
}
