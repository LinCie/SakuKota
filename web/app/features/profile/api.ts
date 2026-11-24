import { apiGet, apiPost, apiPut, unwrap } from "~/shared/api";
import type { MediaUpload } from "~/shared/image-upload";
import type { Profile } from "./types";

// Every function here calls one backend route (see internal/router/router.go)
// on the signed-in user's profile, returns the response data, and throws an
// `ApiError` when the backend rejects the request.

export async function getProfile(): Promise<Profile> {
  return unwrap(await apiGet<Profile>("users/me"));
}

/** Changes the display name; the backend trims it. */
export async function renameMe(name: string): Promise<Profile> {
  return unwrap(await apiPut<Profile>("users/me", { json: { name } }));
}

/** Presigns an upload of a new profile photo; images only. */
export async function requestAvatarUpload(
  contentType: string,
): Promise<MediaUpload> {
  return unwrap(
    await apiPost<MediaUpload>("users/me/avatar/upload-url", {
      json: { content_type: contentType },
    }),
  );
}

/** Makes the uploaded image the profile photo, replacing the old one. */
export async function setAvatar(uploadKey: string): Promise<Profile> {
  return unwrap(
    await apiPut<Profile>("users/me/avatar", {
      json: { upload_key: uploadKey },
    }),
  );
}
