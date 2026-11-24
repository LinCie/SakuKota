import { createFileRoute } from "@tanstack/react-router";

import { SignUpPage } from "~/features/auth/pages/sign-up-page";

export const Route = createFileRoute("/_auth/daftar")({
  head: () => ({ meta: [{ title: "Daftar · SakuKota" }] }),
  component: SignUpRoute,
});

function SignUpRoute() {
  const { kembali } = Route.useSearch();
  return <SignUpPage kembali={kembali} />;
}
