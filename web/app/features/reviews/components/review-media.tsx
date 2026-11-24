import { Dialog } from "@base-ui/react/dialog";
import { X } from "lucide-react";

import { cn } from "~/shared/cn";
import { ImageFrame } from "~/shared/components/image-frame";
import { buttonVariants } from "~/shared/components/ui/button";
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
        className="focus-visible:ring-ring/50 block w-full cursor-zoom-in rounded-xl outline-none focus-visible:ring-3"
      >
        {thumbnail}
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Backdrop className="bg-foreground/80 fixed inset-0 z-50 transition-opacity duration-150 data-ending-style:opacity-0 data-starting-style:opacity-0" />
        <Dialog.Viewport className="fixed inset-0 z-50 grid place-items-center p-4">
          <Dialog.Popup className="relative flex max-h-full max-w-full transition-opacity duration-150 outline-none data-ending-style:opacity-0 data-starting-style:opacity-0">
            <Dialog.Title className="sr-only">{alt}</Dialog.Title>
            <img
              src={media.url}
              alt={alt}
              className="bg-muted max-h-[calc(100dvh-2rem)] max-w-full rounded-xl object-contain"
            />
            <Dialog.Close
              aria-label="Tutup"
              className={cn(
                buttonVariants({ variant: "floating", size: "icon-sm" }),
                "absolute top-2 right-2",
              )}
            >
              <X aria-hidden="true" className="size-5" />
            </Dialog.Close>
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
