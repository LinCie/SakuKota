import { ImageUp, Trash2 } from "lucide-react";
import { useRef } from "react";

import {
  imageAccept,
  imageErrorMessage,
  maxMedias,
  type Media,
} from "~/shared/image-upload";

import { Button } from "./button";
import { FormAlert } from "./form-alert";
import { ImageFrame } from "./image-frame";
import { SectionCard } from "./section-card";

type MediaCardProps = {
  title: string;
  medias: Media[];
  alt: string;
  /** Names what is missing when there are no images. */
  missing: string;
  /** Aspect ratio class of each image, e.g. `aspect-4/3`. */
  aspect: string;
  pending: boolean;
  /** Id of the image being deleted, if any. */
  deletingId: number | null;
  error: Error | null;
  /** Called with the file the admin picked. */
  onAdd: (file: File) => void;
  onDelete: (media: Media) => void;
};

/** A place's or sub-district's images, each with a delete button, and a button to add another. */
export function MediaCard({
  title,
  medias,
  alt,
  missing,
  aspect,
  pending,
  deletingId,
  error,
  onAdd,
  onDelete,
}: MediaCardProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const full = medias.length >= maxMedias;

  return (
    <SectionCard title={title}>
      {medias.length === 0 ? (
        <ImageFrame src={null} alt={alt} missing={missing} aspect={aspect} />
      ) : (
        <ul className="grid grid-cols-2 gap-3">
          {medias.map((media) => (
            <li key={media.id} className="flex flex-col gap-2">
              <ImageFrame
                src={media.url}
                alt={`${alt} ${media.position}`}
                missing={missing}
                aspect={aspect}
              />
              <Button
                className="text-chili"
                disabled={deletingId === media.id}
                onClick={() => onDelete(media)}
              >
                <Trash2
                  aria-hidden="true"
                  className="size-4"
                  strokeWidth={2.5}
                />
                {deletingId === media.id ? "Menghapus…" : "Hapus"}
              </Button>
            </li>
          ))}
        </ul>
      )}
      {error ? <FormAlert>{imageErrorMessage(error)}</FormAlert> : null}
      <div className="flex flex-wrap items-center gap-3">
        <Button
          disabled={pending || full}
          onClick={() => inputRef.current?.click()}
        >
          <ImageUp aria-hidden="true" className="size-4" strokeWidth={2.5} />
          {pending ? "Mengunggah…" : "Tambah gambar"}
        </Button>
        <p className="text-ink-soft text-xs">
          {full
            ? `Sudah ${maxMedias} gambar. Hapus satu untuk menambah.`
            : `JPEG, PNG, atau WebP, maks. 5 MB. ${medias.length}/${maxMedias} gambar.`}
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
          if (file) onAdd(file);
        }}
      />
    </SectionCard>
  );
}
