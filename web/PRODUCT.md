# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Fresh graduates, trainees, and young professionals who have just relocated to Jakarta for work (persona: Dimas, 23, corporate trainee moved from Surabaya to Setiabudi). They are new to the neighborhood, budget-conscious, and most often on a phone after work, looking for a nearby affordable meal, laundry, tailor, clinic, or pharmacy, and wanting to know whether their kecamatan floods, how safe it is, and what a day costs.

## Product Purpose

Rantau is a hyper-local survival map for young perantau in Jakarta. It surfaces curated, budget-friendly daily essentials that standard map platforms bury or get wrong, alongside practical neighborhood insight per kecamatan. Success: a newcomer finds a suitable nearby essential and understands their area's flood, safety, and spending picture without trial-and-error or asking random locals.

## Positioning

Curated micro-services (warung, tailors, local clinics, laundries, pharmacies) rather than the restaurants and major businesses general map apps favor, combined with kecamatan-level flood history, safety/crime notes, and estimated daily spending in one place.

## Operating Context

- Primary scenario: mobile, late after work, filtering nearby pins by category, tapping for details, and checking the kecamatan info.
- Also used on desktop.
- Single-screen app flow: pick kecamatan, filter/search, browse map pins, open place details, open kecamatan info.
- Optional account: "Masuk" in the header opens `/masuk` or `/daftar` and returns to the same map view afterwards. The map never requires an account.

## Capabilities and Constraints

- Frontend: TanStack Router (file routes) + Tailwind CSS v4 in `web/`; Tailwind utilities only, no raw CSS.
- Map: Leaflet (confirmed) with real coordinates from the API.
- Backend (Go, PostGIS) endpoints: `GET /places/` (optional `sub_district_id`, comma-separated `category_id`, and `q` search), `GET /places/categories/`, `GET /subdistricts/`.
- Categories: Affordable Food & Warung, Tailor, Local Clinic, Laundry, Pharmacy.
- Place fields: name, address, location (GeoJSON point), operating hours, estimated price range, short description. Nullable fields must show placeholders ("Hours not available", "Data not available").
- Kecamatan fields: flood history, safety/crime information, estimated daily spending (Rupiah), static map image URL.
- Auth endpoints: `POST /auth/register` (name, email, password of at least 8 characters; returns the user, no tokens), `POST /auth/login` (user plus access and refresh tokens), `POST /auth/refresh` (single-use refresh tokens, held in backend memory, so a restart ends sessions), `POST /auth/logout`. No endpoint requires sign-in yet.
- Accounts: the map is public; accounts are groundwork for later features and must not be advertised with benefits that don't exist. Registering signs the user in automatically. The session (user plus tokens) lives in localStorage; the return path travels in the `kembali` search param.
- Terminology: "kecamatan" in the UI (backend calls it sub-district).
- Out of scope: admin verification, place suggestions, votes/reviews, GeoJSON overlays, kost listings, password reset, social login.
- UI language: Bahasa Indonesia (confirmed). Seed content is currently English; translating it is a separate data decision.

## Brand Commitments

- Name: Rantau.
- Layout reference: the low-fi wireframe in the planning docs (not published) (header with kecamatan picker, View Info, and search; category filter bar; result count; map with place detail panel; kecamatan info modal). The homepage layout should stay similar to it.

## Evidence on Hand

- Seed data in `../cmd/seed/seed.sql`: 2 kecamatan (Setiabudi, Mampang Prapatan), about 10 curated places each across the five categories. Static map images are placeholders (placehold.co).
- No testimonials, user counts, or partner claims exist; do not fabricate them.

## Product Principles

- Practical first: every screen helps a newcomer decide where to go or whether an area suits them.
- Curated over comprehensive: fewer, trustworthy essentials beat an exhaustive listing.
- Honest about gaps: missing data is shown as missing, never hidden or invented.
- Phone-first under real conditions: usable one-handed, late, on the move.
