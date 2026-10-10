import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [
    tailwindcss(),
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
      routesDirectory: "./app/routes",
      generatedRouteTree: "./app/routeTree.gen.ts",
    }),
    react(),
  ],
  resolve: {
    tsconfigPaths: true,
  },
  server: {
    proxy: {
      // Serve the API from the app's own origin so the HttpOnly refresh
      // cookie is first-party.
      "/api": {
        target: "http://localhost:8080/v1",
        rewrite: (path) => path.replace(/^\/api/, ""),
        // The backend scopes the refresh cookie to /v1/auth; here the auth
        // routes are at /api/auth.
        cookiePathRewrite: { "/v1/auth": "/api/auth" },
      },
    },
  },
});
