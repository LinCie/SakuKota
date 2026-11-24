# SakuKota web

The React frontend for SakuKota: a public survival map of curated daily essentials in Jakarta, where signed-in users can review places and suggest new ones, plus an admin area for managing places and reviewing suggestions. Categories are defined by the backend and are read-only here; kecamatan have no admin screens yet. See [`PRODUCT.md`](PRODUCT.md) for the product, [`DESIGN.md`](DESIGN.md) for the visual system, and [`../docs/frontend.md`](../docs/frontend.md) for routes, components, and state.

## Stack

React 19, TypeScript, Vite, TanStack Router (file routes in `app/routes`), TanStack Query and Form, Zod, ky, Base UI, Tailwind CSS v4 (utility classes only, no raw CSS), Leaflet, Vitest, and pnpm.

## Development

The API must be running (see the [root README](../README.md)); Vite proxies `/api` to `http://localhost:8080/v1`.

```bash
pnpm install
pnpm dev            # http://localhost:5173
pnpm typecheck      # tsc
pnpm lint           # ESLint
pnpm test           # Vitest
pnpm format:check   # Prettier
pnpm build          # production build in dist/
```

## Layout

```
app/routes/      file-based routes: the map (/), sign-in (/masuk, /daftar), suggestions (/usulan), and /admin
app/features/    one folder per domain: places, subdistricts, reviews, suggestions, auth, admin
app/shared/      API client, hooks, and shared components
```

Categories come from the API with their `code`, `label`, and `color`. The `color` is a palette key (`chili`, `plum`, `cat`, `sign`, `board`) that `app/features/places/components/category-sign.tsx` maps to Tailwind classes.
