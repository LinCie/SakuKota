import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { getCategory } from "~/features/places/api";
import { CategoryDetailPage } from "~/features/categories/components/category-detail-page";
import { LoadingState } from "~/shared/components/loading-state";
import { unwrap } from "~/shared/unwrap";

export const Route = createFileRoute("/admin/categories/$categoryId")({
  loader: async ({ params }) => ({
    category: unwrap(await getCategory(Number(params.categoryId))),
  }),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  component: CategoryDetailRoute,
});

function CategoryDetailRoute() {
  return <CategoryDetailPage {...Route.useLoaderData()} />;
}
