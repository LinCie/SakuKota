# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Fresh graduates, trainees, and young professionals who have just relocated to Jakarta for work (persona: Dimas, 23, corporate trainee moved from Surabaya to Setiabudi). They are new to the neighborhood, budget-conscious, and most often on a phone after work, looking for a nearby affordable meal, laundry, tailor, clinic, or pharmacy, and wanting to know whether their kecamatan floods, how safe it is, and what a day costs.

## Product Purpose

Rantau is a hyper-local survival map for young perantau in Jakarta. It surfaces curated, budget-friendly daily essentials that standard map platforms bury or get wrong, alongside practical neighborhood insight per kecamatan. Success: a newcomer finds a suitable nearby essential and understands their area's flood, safety, and spending picture without trial-and-error or asking random locals.

## Positioning

Curated micro-services (warung, tailors, local clinics, laundries, pharmacies, plus hospitals, mosques, police posts, and other perantau essentials) rather than the restaurants and major businesses general map apps favor, combined with kecamatan-level flood history, safety/crime notes, and estimated daily spending in one place.

## Operating Context

- Primary scenario: mobile, late after work, filtering nearby pins by category, tapping for details, and checking the kecamatan info.
- Also used on desktop.
- Single-screen app flow: pick kecamatan, filter/search, browse map pins, open place details and reviews, open kecamatan info.
- Optional account: "Masuk" in the header opens `/masuk` or `/daftar` and returns to the same map view afterwards. The map never requires an account. Signed in, a user can review a place from its details and suggest a place at `/usulan`, where they also follow their suggestions.
- Admins manage places and review suggestions in `/admin`.

## Capabilities and Constraints

- Frontend: TanStack Router (file routes) + Tailwind CSS v4 in `web/`; Tailwind utilities only, no raw CSS.
- Map: Leaflet (confirmed) with real coordinates from the API.
- Map: it cannot zoom out past zoom 13 (a whole kecamatan on a phone) and does not cluster pins. It loads the places nearest the centre of its view, 100 at zoom 13 down to 40 from zoom 16; the list beside it shows every match in the kecamatan, 20 at a time. Each kecamatan's boundary is outlined; clicking another outline picks it.
- Backend (Go, PostGIS) endpoints: `GET /places/` (optional `sub_district_id`, comma-separated `category_id`, `q` search, `bbox` as `west,south,east,north`, `limit` up to 100, and `offset`; returns `{places, total}`, where `total` counts every match before the limit), `GET /places/categories/`, `GET /subdistricts/`.
- Categories: Affordable Food & Warung, Tailor, Local Clinic, Laundry, Pharmacy, Mosque & Musala, Other Worship, Police Post, Hospital, Community Health Center, Bank & ATM, Minimarket, Transit Stop, Public Service Office, Water & Gas Refill, Print & Copy.
- Place fields: name, address, location (GeoJSON point), operating hours, estimated price range, short description, average rating and rating count, up to 10 photos or videos (`medias`), plus custom fields per category (`attributes`, described by each category's `fields` from `GET /places/categories/`; data types text, number, boolean, select, range, date). Nullable fields must show placeholders ("Hours not available", "Data not available").
- Kecamatan fields: flood history, safety/crime information, estimated daily spending (Rupiah), up to 10 photos or videos (`medias`), boundary (GeoJSON MultiPolygon, or null).
- Reviews: `GET /places/{id}/reviews` (public) and `POST /places/{id}/reviews` (signed in): 1 to 5 stars with an optional comment, one review per user per place, no editing or deleting.
- Suggestions: `POST /suggestions` and `GET /suggestions/mine` (signed in; at most 5 pending per user). Admins list, approve (with corrected place details), or reject (with an optional reason) them.
- Auth endpoints: `POST /auth/register` (name, email, password of at least 8 characters; returns the user, no tokens), `POST /auth/login` (user plus access token; the refresh token is an HttpOnly cookie), `POST /auth/refresh` (single-use refresh tokens, kept in Redis, so sessions survive a backend restart), `POST /auth/logout`. Reviewing and suggesting require sign-in; place, kecamatan, and suggestion-review writes require the `admin` role. The frontend's `/admin` area covers places (with their photos) and suggestions. It has no kecamatan screens, and categories are read-only because the backend defines them.
- Accounts: the map is public; an account lets a user review places and suggest new ones, and must not be advertised with benefits beyond those. Registering signs the user in automatically. The session (user plus access token) lives in memory only and is restored from the refresh cookie on page load; the return path travels in the `kembali` search param.
- Images: admins upload JPEG, PNG, or WebP up to 5 MB and MP4 or WebM up to 50 MB, straight to storage through a presigned link.
- Terminology: "kecamatan" in the UI (backend calls it sub-district).
- Out of scope: up/down votes, kost listings, password reset, social login.
- UI language: Bahasa Indonesia (confirmed). Seed content is currently English; translating it is a separate data decision.

## Brand Commitments

- Name: Rantau.
- Layout reference: the low-fi wireframe in the planning docs (not published) (header with kecamatan picker, View Info, and search; category filter dropdown; result count; map with place detail panel; kecamatan info modal). The homepage layout should stay similar to it.

## Evidence on Hand

- Seed data in `../cmd/seed/seed.sql`: 3 kecamatan (Setiabudi, Mampang Prapatan, Tebet), about 1,070 real places from OpenStreetMap with made-up details filled in, 24 made-up water and gas refills, 12 demo users, and about 1,070 made-up reviews. `../cmd/seed-bulk` holds 50,000 generated places for load testing. Place and kecamatan photos are generated single-color placeholders; the kecamatan boundaries are their OpenStreetMap administrative boundaries.
- No testimonials, user counts, or partner claims exist; do not fabricate them.

## Product Principles

- Practical first: every screen helps a newcomer decide where to go or whether an area suits them.
- Curated over comprehensive: fewer, trustworthy essentials beat an exhaustive listing.
- Honest about gaps: missing data is shown as missing, never hidden or invented.
- Phone-first under real conditions: usable one-handed, late, on the move.
