import type { AdminWriteErrorCode } from "./api";
import { writeErrorMessage } from "./write-error";

// -----------------------------------------------------------------------------
// Images are uploaded straight to object storage: the backend presigns an
// upload, the browser sends the file to it, and the backend then confirms the
// upload key (see internal/storage/media_store.go).
// -----------------------------------------------------------------------------

/** A presigned upload (see MediaUploadResponse in the backend's dto.go files). */
export type MediaUpload = {
  /** Key to confirm once the file is uploaded. */
  upload_key: string;
  upload_url: string;
  method: string;
  /** Headers to send with the upload exactly as given. */
  headers: Record<string, string>;
  expires_at: string;
};

/** An image of a place or sub-district (see Media in the backend's storage/dto.go). */
export type Media = {
  id: number;
  url: string;
  content_type: string;
  /** Size in bytes. */
  size: number;
  /** 1 to 10, in display order. */
  position: number;
};

/** Most images a place or sub-district can have. */
export const maxMedias = 10;

/** Error codes for requesting or confirming an image upload. */
export type ImageErrorCode =
  AdminWriteErrorCode | "VALIDATION_ERROR" | "MEDIA_LIMIT_REACHED";

/** Content types the backend accepts, as an `accept` attribute value. */
export const imageAccept =
  "image/jpeg,image/png,image/webp,video/mp4,video/webm";

/** Largest image the backend accepts, in bytes. */
const maxImageSize = 5 * 1024 * 1024;

/** Largest video the backend accepts, in bytes. */
const maxVideoSize = 50 * 1024 * 1024;

/** Thrown when a file is rejected before upload or storage refuses it. */
export class ImageUploadError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ImageUploadError";
  }
}

/**
 * Checks `file`, presigns an upload for it with `presign`, and sends it to
 * storage. Resolves to the upload key, for the caller to confirm.
 */
export async function uploadImage(
  file: File,
  presign: (contentType: string) => Promise<MediaUpload>,
): Promise<string> {
  if (!imageAccept.split(",").includes(file.type)) {
    throw new ImageUploadError(
      "Pilih gambar JPEG, PNG, atau WebP, atau video MP4 atau WebM.",
    );
  }
  if (file.type.startsWith("video/")) {
    if (file.size > maxVideoSize) {
      throw new ImageUploadError("Ukuran video maksimal 50 MB.");
    }
  } else if (file.size > maxImageSize) {
    throw new ImageUploadError("Ukuran gambar maksimal 5 MB.");
  }

  const upload = await presign(file.type);

  // Plain fetch, not the API client: the URL carries its own signature, which
  // the client's bearer token would break.
  const response = await fetch(upload.upload_url, {
    method: upload.method,
    headers: upload.headers,
    body: file,
  });
  if (!response.ok) {
    throw new ImageUploadError("File gagal diunggah. Coba lagi.");
  }

  return upload.upload_key;
}

/** A message for a failed image upload. */
export function imageErrorMessage(error: Error): string {
  return error instanceof ImageUploadError
    ? error.message
    : writeErrorMessage(error);
}
