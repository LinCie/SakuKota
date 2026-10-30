import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { CategoriesPage } from "~/features/categories/pages/categories-page";
import { categoriesQuery, useCategories } from "~/features/places/queries";
import { LoadingState } from "~/shared/components/loading-state";

export const Route = createFileRoute("/admin/categories/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(categoriesQuery()),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  head: () => ({ meta: [{ title: "Kategori · Admin Rantau" }] }),
  component: CategoriesRoute,
});

function CategoriesRoute() {
  const { data: categories } = useCategories();
  return <CategoriesPage categories={categories} />;
}
