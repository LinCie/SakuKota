import { createFileRoute } from "@tanstack/react-router";

import { AdminRouteError } from "~/features/admin/components/admin-route-error";
import { CategoryDetailPage } from "~/features/categories/pages/category-detail-page";
import { categoryQuery, useCategory } from "~/features/places/queries";
import { LoadingState } from "~/shared/components/loading-state";

export const Route = createFileRoute("/admin/categories/$categoryId")({
  loader: ({ context: { queryClient }, params }) =>
    queryClient.ensureQueryData(categoryQuery(Number(params.categoryId))),
  pendingComponent: LoadingState,
  errorComponent: AdminRouteError,
  component: CategoryDetailRoute,
});

function CategoryDetailRoute() {
  const { categoryId } = Route.useParams();
  const { data: category } = useCategory(Number(categoryId));
  return <CategoryDetailPage category={category} />;
}
