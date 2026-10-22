import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { getCategories } from "~/features/places/api";
import { CategoriesPage } from "~/features/categories/components/categories-page";
import { LoadingState } from "~/shared/components/loading-state";
import { unwrap } from "~/shared/api";

export const Route = createFileRoute("/admin/categories/")({
  loader: async () => ({ categories: unwrap(await getCategories()) }),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Kategori · Admin Rantau" }] }),
  component: CategoriesRoute,
});

function CategoriesRoute() {
  return <CategoriesPage {...Route.useLoaderData()} />;
}
