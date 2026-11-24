import { createFileRoute, Navigate, redirect } from "@tanstack/react-router";

import {
  getSession,
  restoreSession,
  useSession,
} from "~/features/auth/session";
import {
  ProfileError,
  ProfilePage,
} from "~/features/profile/pages/profile-page";
import { profileQuery } from "~/features/profile/queries";
import { myReviewsQuery } from "~/features/reviews/queries";

export const Route = createFileRoute("/profil")({
  beforeLoad: async ({ location }) => {
    await restoreSession();
    const session = getSession();
    if (!session) {
      throw redirect({
        to: "/masuk",
        search: { kembali: location.href },
        replace: true,
      });
    }
    return { userId: session.user.id };
  },
  loader: ({ context: { queryClient, userId } }) =>
    Promise.all([
      queryClient.query(profileQuery(userId)),
      queryClient.query(myReviewsQuery(userId)),
    ]),
  head: () => ({ meta: [{ title: "Profil · SakuKota" }] }),
  component: ProfileRoute,
  errorComponent: ProfileError,
});

function ProfileRoute() {
  const session = useSession();
  // Signing out here leaves the page for the map.
  if (!session) return <Navigate to="/" replace />;
  return <ProfilePage userId={session.user.id} />;
}
