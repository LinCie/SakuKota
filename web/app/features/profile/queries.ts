import {
  queryOptions,
  useMutation,
  useQueryClient,
  useSuspenseQuery,
  type QueryClient,
} from "@tanstack/react-query";

import { getSession, saveSession } from "~/features/auth/session";
import { reviewKeys } from "~/features/reviews/queries";
import { ImageUploadError, uploadImage } from "~/shared/image-upload";

import { getProfile, renameMe, requestAvatarUpload, setAvatar } from "./api";
import type { Profile } from "./types";

export const profileKeys = {
  /** Keyed by user, so one user's profile never shows for the next. */
  me: (userId: number) => ["profile", userId] as const,
};

/** Profile photos the backend accepts, as an `accept` attribute value. */
export const avatarAccept = "image/jpeg,image/png,image/webp";

// Queries. The options are shared by the route loader (`query`) and the hook
// below, which reads the data the loader put in the cache.

export function profileQuery(userId: number) {
  return queryOptions({
    queryKey: profileKeys.me(userId),
    queryFn: getProfile,
  });
}

export function useProfile(userId: number) {
  return useSuspenseQuery(profileQuery(userId));
}

// Mutations. Each resolves once the reviews, which show the author's name and
// photo, refetched.

export function useRename() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: renameMe,
    onSuccess: (profile) => profileChanged(queryClient, profile),
  });
}

export function useSetAvatar() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (file: File) => {
      // uploadImage also takes videos, which a profile photo cannot be.
      if (!avatarAccept.split(",").includes(file.type)) {
        throw new ImageUploadError("Pilih gambar JPEG, PNG, atau WebP.");
      }
      return setAvatar(await uploadImage(file, requestAvatarUpload));
    },
    onSuccess: (profile) => profileChanged(queryClient, profile),
  });
}

function profileChanged(queryClient: QueryClient, profile: Profile) {
  queryClient.setQueryData(profileKeys.me(profile.id), profile);
  // The header shows the session's name.
  const session = getSession();
  if (session?.user.id === profile.id) {
    saveSession({ ...session, user: { ...session.user, name: profile.name } });
  }
  return queryClient.invalidateQueries({ queryKey: reviewKeys.all });
}
