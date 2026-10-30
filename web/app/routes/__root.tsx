import type { QueryClient } from "@tanstack/react-query";
import {
  createRootRouteWithContext,
  HeadContent,
  Outlet,
  type ErrorComponentProps,
} from "@tanstack/react-router";

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
}>()({
  component: RootLayout,
  errorComponent: RootError,
  notFoundComponent: NotFound,
});

function RootLayout() {
  return (
    <>
      <HeadContent />
      <Outlet />
    </>
  );
}

function NotFound() {
  return (
    <main className="container mx-auto p-4 pt-16">
      <h1>404</h1>
      <p>The requested page could not be found.</p>
    </main>
  );
}

function RootError({ error }: ErrorComponentProps) {
  const devError =
    import.meta.env.DEV && error instanceof Error ? error : undefined;

  return (
    <main className="container mx-auto p-4 pt-16">
      <h1>Oops!</h1>
      <p>{devError?.message ?? "An unexpected error occurred."}</p>
      {devError?.stack ? (
        <pre className="w-full overflow-x-auto p-4">
          <code>{devError.stack}</code>
        </pre>
      ) : null}
    </main>
  );
}
