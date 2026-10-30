# Rantau web

The React frontend for Rantau: a public survival map of curated daily essentials in Jakarta, plus an admin area for managing places and sub-districts (categories are viewable but read-only, since the backend defines them). See [`PRODUCT.md`](PRODUCT.md) for the product and [`DESIGN.md`](DESIGN.md) for the visual system.

## Stack

React 19, TypeScript, Vite, TanStack Router (file routes in `app/routes`), TanStack Query and Form, Tailwind CSS v4 (utility classes only, no raw CSS), Leaflet, and pnpm.

## Development

The API must be running (see the [root README](../README.md)); Vite proxies `/api` to `http://localhost:8080/v1`.

```bash
pnpm install
pnpm dev            # http://localhost:5173
pnpm typecheck      # tsc
pnpm lint           # ESLint
pnpm format:check   # Prettier
pnpm build          # production build in dist/
```

## Layout

```
app/routes/      file-based routes: the map (/), sign-in (/masuk, /daftar), and /admin
app/features/    one folder per domain: places, subdistricts, categories, auth, admin
app/shared/      API client, hooks, and shared components
```

Categories come from the API with their `code`, `label`, and `color`. The `color` is a palette key (`chili`, `plum`, `cat`, `sign`, `board`) that `app/features/places/components/category-sign.tsx` maps to Tailwind classes.
