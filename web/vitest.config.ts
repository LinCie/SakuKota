import { defineConfig } from "vitest/config";

// Separate from vite.config.ts: tests don't need its router, Tailwind, or
// React plugins (the router plugin would regenerate the route tree).
export default defineConfig({
  resolve: {
    tsconfigPaths: true,
  },
  test: {
    include: ["app/**/*.test.ts"],
    // An absolute URL, since there is no page origin to resolve "/api" against.
    env: { VITE_API_URL: "http://backend.test/api" },
  },
});
