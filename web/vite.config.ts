import babel from "@rolldown/plugin-babel";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => {
  // Dev-server settings only; the unprefixed names keep them out of the
  // client bundle. Variables from the environment (`make dev`) override .env.
  const env = loadEnv(mode, process.cwd(), "");
  const apiTarget = (env.API_PROXY_TARGET || "http://localhost:8080").replace(
    /\/+$/,
    "",
  );

  return {
    plugins: [
      tailwindcss(),
      tanstackRouter({
        target: "react",
        autoCodeSplitting: true,
        routesDirectory: "./app/routes",
        generatedRouteTree: "./app/routeTree.gen.ts",
      }),
      react(),
      babel({ presets: [reactCompilerPreset()] }),
    ],
    resolve: {
      tsconfigPaths: true,
    },
    server: {
      // A taken port fails instead of moving, since the image bucket's CORS rule
      // allows only CORS_ALLOWED_ORIGIN.
      port: Number(env.DEV_PORT) || 5173,
      strictPort: true,
      proxy: {
        // Serve the API from the app's own origin so the HttpOnly refresh
        // cookie is first-party.
        "/api": {
          target: `${apiTarget}/v1`,
          rewrite: (path) => path.replace(/^\/api/, ""),
          // The backend scopes the refresh cookie to /v1/auth; here the auth
          // routes are at /api/auth.
          cookiePathRewrite: { "/v1/auth": "/api/auth" },
        },
      },
    },
  };
});
