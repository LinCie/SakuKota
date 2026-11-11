import { createFileRoute, Navigate, redirect } from "@tanstack/react-router";

import { useSession } from "~/features/auth/hooks";
import { getSession, restoreSession } from "~/features/auth/session";
import { ensurePlaceData, usePlaceData } from "~/features/places/queries";
import {
  SuggestError,
  SuggestPage,
} from "~/features/suggestions/pages/suggest-page";
import {
  mySuggestionsQuery,
  useMySuggestions,
} from "~/features/suggestions/queries";

export const Route = createFileRoute("/usulan")({
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
  // The places are for the form's map, which shows what a kecamatan already has.
  loader: ({ context: { queryClient, userId } }) =>
    Promise.all([
      ensurePlaceData(queryClient),
      queryClient.query({
        ...mySuggestionsQuery(userId),
        staleTime: "static",
      }),
    ]),
  head: () => ({ meta: [{ title: "Usulkan tempat · Rantau" }] }),
  component: SuggestRoute,
  errorComponent: SuggestError,
});

function SuggestRoute() {
  const session = useSession();
  // Signing out here leaves the page for the map.
  if (!session) return <Navigate to="/" replace />;
  return <SuggestContent userId={session.user.id} />;
}

function SuggestContent({ userId }: { userId: number }) {
  const { data: suggestions } = useMySuggestions(userId);
  return <SuggestPage suggestions={suggestions} {...usePlaceData()} />;
}
