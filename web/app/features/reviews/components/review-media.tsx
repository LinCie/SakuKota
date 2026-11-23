import { Dialog } from "@base-ui/react/dialog";
import { X } from "lucide-react";

import { ImageFrame } from "~/shared/components/image-frame";
import type { Media } from "~/shared/image-upload";

type ReviewMediaProps = { media: Media; alt: string };

/** A review's image as a thumbnail that opens full size in a modal. Videos play in place. */
export function ReviewMedia({ media, alt }: ReviewMediaProps) {
  const thumbnail = (
    <ImageFrame
      src={media.url}
      contentType={media.content_type}
      alt={alt}
      missing="Foto tidak tersedia"
      aspect="aspect-square"
    />
  );

  // A video has its own controls, which a button around it would swallow.
  if (media.content_type.startsWith("video/")) return thumbnail;

  return (
    <Dialog.Root>
      <Dialog.Trigger
        aria-label={`Perbesar ${alt}`}
        className="focus-visible:outline-cat block w-full cursor-zoom-in rounded-[3px] focus-visible:outline-2 focus-visible:outline-offset-2"
      >
        {thumbnail}
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Backdrop className="bg-ink/80 fixed inset-0 z-50 transition-opacity duration-150 data-ending-style:opacity-0 data-starting-style:opacity-0" />
        <Dialog.Viewport className="fixed inset-0 z-50 grid place-items-center p-4">
          <Dialog.Popup className="relative flex max-h-full max-w-full transition-opacity duration-150 outline-none data-ending-style:opacity-0 data-starting-style:opacity-0">
            <Dialog.Title className="sr-only">{alt}</Dialog.Title>
            <img
              src={media.url}
              alt={alt}
              className="border-ink bg-paint-deep max-h-[calc(100dvh-2rem)] max-w-full rounded-[3px] border-2 object-contain"
            />
            <Dialog.Close
              aria-label="Tutup"
              className="bg-paint/90 text-ink border-ink hover:bg-paint focus-visible:outline-cat absolute top-2 right-2 grid size-9 place-items-center rounded-[3px] border-2 focus-visible:outline-2"
            >
              <X aria-hidden="true" className="size-5" strokeWidth={2.5} />
            </Dialog.Close>
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
