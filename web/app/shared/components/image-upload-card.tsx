import { ImageUp } from "lucide-react";
import { useRef } from "react";

import { imageAccept, imageErrorMessage } from "~/shared/image-upload";

import { Button } from "./button";
import { FormAlert } from "./form-alert";
import { ImageFrame } from "./image-frame";
import { SectionCard } from "./section-card";

type ImageUploadCardProps = {
  title: string;
  imageUrl: string | null;
  alt: string;
  /** Names what is missing when there is no image. */
  missing: string;
  /** Aspect ratio class of the image, e.g. `aspect-8/5`. */
  aspect: string;
  pending: boolean;
  error: Error | null;
  /** Called with the file the admin picked. */
  onUpload: (file: File) => void;
};

/** An image with a button to upload a new one in its place. */
export function ImageUploadCard({
  title,
  imageUrl,
  alt,
  missing,
  aspect,
  pending,
  error,
  onUpload,
}: ImageUploadCardProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <SectionCard title={title}>
      <ImageFrame src={imageUrl} alt={alt} missing={missing} aspect={aspect} />
      {error ? <FormAlert>{imageErrorMessage(error)}</FormAlert> : null}
      <div className="flex flex-wrap items-center gap-3">
        <Button disabled={pending} onClick={() => inputRef.current?.click()}>
          <ImageUp aria-hidden="true" className="size-4" strokeWidth={2.5} />
          {pending
            ? "Mengunggah…"
            : imageUrl
              ? "Ganti gambar"
              : "Unggah gambar"}
        </Button>
        <p className="text-ink-soft text-xs">
          JPEG, PNG, atau WebP, maks. 5 MB.
        </p>
      </div>
      <input
        ref={inputRef}
        type="file"
        accept={imageAccept}
        hidden
        onChange={(event) => {
          const file = event.target.files?.[0];
          // Cleared so picking the same file again still uploads it.
          event.target.value = "";
          if (file) onUpload(file);
        }}
      />
    </SectionCard>
  );
}
