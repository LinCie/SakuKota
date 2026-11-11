import { createFileRoute } from "@tanstack/react-router";

import { SignInPage } from "~/features/auth/pages/sign-in-page";

export const Route = createFileRoute("/_auth/masuk")({
  head: () => ({ meta: [{ title: "Masuk · Rantau" }] }),
  component: SignInRoute,
});

function SignInRoute() {
  const { kembali } = Route.useSearch();
  return <SignInPage kembali={kembali} />;
}
