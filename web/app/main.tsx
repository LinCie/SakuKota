import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import {
  getAccessToken,
  refreshSession,
  restoreSession,
} from "~/features/auth/session";
import { setAuthHandlers } from "~/shared/api";

import { routeTree } from "./routeTree.gen";
import "@fontsource/bungee/latin-400.css";
import "@fontsource-variable/figtree";
import "./app.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // ky already retries network failures, and backend errors won't change.
      retry: false,
      // Loaders fill the cache right before a page mounts; a short stale time
      // keeps the page from refetching on mount, while mutations invalidate
      // what they change.
      staleTime: 30_000,
    },
    mutations: { retry: false },
  },
});

const router = createRouter({
  routeTree,
  context: { queryClient },
  scrollRestoration: true,
  // TanStack Query owns caching, so preloads always run the loaders.
  defaultPreloadStaleTime: 0,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

// Plug the session into the shared API client. The session is kept in memory
// only, so restore it from the refresh cookie. Rendering doesn't wait for it:
// the header updates when it lands, and route guards await it.
setAuthHandlers({ getAccessToken, refresh: refreshSession });
void restoreSession();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
);
