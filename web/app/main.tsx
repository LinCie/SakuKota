import { createRouter, RouterProvider } from "@tanstack/react-router";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { getAccessToken, refreshSession } from "~/features/auth/session";
import { setAuthHandlers } from "~/shared/api";

import { routeTree } from "./routeTree.gen";
import "@fontsource/bungee/latin-400.css";
import "@fontsource-variable/figtree";
import "./app.css";

const router = createRouter({ routeTree, scrollRestoration: true });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

// Plug the session into the shared API client. The session is kept in memory
// only, so restore it from the refresh cookie before the first render; route
// guards and the header then see the user.
setAuthHandlers({ getAccessToken, refresh: refreshSession });
await refreshSession();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
);
