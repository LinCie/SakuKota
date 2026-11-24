# SakuKota — Project Context

> **Living document.** What the project is for and what the backend is specified
> to do. Update it in the same task as any verified change to backend behavior or
> the API contract (see [Maintaining this file](#maintaining-this-file)).
>
> - Last updated: 2026-11-25 (request context timeout).
> - Status: all Go unit and integration tests pass. `web/` is untested.

## 1. Why the project exists

The original planning docs (not published) call it **Perantau Helper**; here it
is **SakuKota** / `sakukota-go`.

- **Problem.** People who move to Jakarta for work (*perantau*) struggle to find
  cheap everyday essentials (warung, laundry, clinics, pharmacies, mosques,
  transit, ATMs, public offices, …) and know little about each neighbourhood's
  flood risk, safety, and cost of living.
- **Users.** Fresh graduates, trainees, and young professionals. Persona: Dimas,
  23, a trainee in Setiabudi, using the app on his phone after work.
- **Value.** A hyper-local "survival map": curated essentials plus safety and
  cost information per kecamatan.
- **Non-goal.** Kost / boarding-house listings. Do not add them.
- **Quality stance.** Admin-verified data over volume; placeholders over broken
  layouts when data is missing.

## 2. Glossary

| Term | Meaning |
| ---- | ------- |
| Kecamatan / sub-district | Jakarta district; `subdistrict` / `sub_district` in code and API |
| Place / pin | A curated spot on the map |
| Category | Place type (e.g. Warung), defined in migrations with `code`, `label`, `color` (palette key) and custom **fields** |
| Attributes | A place's values for its category's fields (JSON object) |
| Suggestion | A user-proposed place awaiting admin review |
| Review | A user's 1–5 star rating of a place, with optional comment and medias |
| Report | A user's flag of an abusive place or review, with a reason, awaiting an admin |

## 3. Sources of truth

1. **Tests (`*_test.go`)** are the executable specification. Production code is
   never evidence of intent.
2. **This file** summarizes intent and the tested contract. If it disagrees with
   a test, the test wins.
3. **Design docs** (the original planning docs, not published) explain the original *why* but
   are partly outdated (§9).

## 4. Scope

| Tier | Feature | Status |
| ---- | ------- | ------ |
| Must | Survival map and pins | Delivered: places API with filters, `bbox`, `limit`, `total` |
| Must | Kecamatan dropdown and info modal | Delivered: sub-districts API with medias and a GeoJSON boundary |
| Must | Category filter and search | Delivered: `category_id` (any of), `q` |
| Should | Admin verification panel | Backend delivered (admin CRUD, suggestion review) |
| Could | Place suggestion form | Delivered, with a pending limit |
| Could | Up/down votes | Delivered as 1–5 star reviews |
| Could | Kecamatan GeoJSON overlay | Delivered; the web map outlines and frames kecamatan |
| Won't | Kost listings | Absent |

Beyond the docs: JWT auth with rotating refresh cookies, up to 10 medias per
place, sub-district, and review in S3-compatible storage (Garage), a cleanup queue
for deleted objects, and abuse reports of places and reviews. Only the Must features have user stories; the rest are
specified only by tests.

## 5. Cross-cutting API contract

Applies to every module unless §6 says otherwise. All routes live under `/v1`.

- **Envelopes.** Success: `{"data": ...}`. Error: `{"code": "UPPER_SNAKE",
  "message": "..."}`, JSON.
- **Status codes.** Create `201`, read/update/action `200`, delete `204` (empty).
- **500s** are always `INTERNAL_ERROR` and never leak driver/SQL details.
- **Path IDs** must be positive int64s, else `400 INVALID_<RESOURCE>_ID` before
  the service is called.
- **`INVALID_BODY` vs `VALIDATION_ERROR`.** An undecodable body (empty, malformed,
  not an object, `null`, trailing data, wrong JSON types, wrong GeoJSON type,
  coordinates outside WGS84) is `400 INVALID_BODY` and never reaches the service.
  A decodable body that breaks a business rule is `400 VALIDATION_ERROR`.
- **Ignored fields.** `id`, `created_at`, `user_id`, `status`, review fields, and
  unknown keys are ignored. The author comes from the token, the ID from the path.
- **PUT** takes the same body as create and replaces every editable field: absent
  or `null` nullable fields are cleared; missing required fields are
  `VALIDATION_ERROR`. A failed update changes nothing.
- **Text.** Inputs are trimmed (inner whitespace kept). Blank optional text is
  stored as `null`. Length limits count Unicode code points after trimming.
- **Length limits** (`VALIDATION_ERROR` when exceeded): user name 100, email 255;
  sub-district name/slug 100; place/suggestion name 150, address 255, operating
  hours 255, price range 100; suggestion note, rejection reason, review comment,
  report note 1000. Descriptions and sub-district free text are unlimited.
- **Locations** are GeoJSON Points `[lon, lat]` inside the **Indonesia box**
  (lon 95–141, lat −11–6, edges included) and round-trip exactly.
- **JSON fidelity.** Attributes round-trip exactly (big ints, `1.50`, any text).
- **Context.** Every service call gets the request context; a cancelled request
  never produces a success. The request context expires after 25 s (under
  the 30 s write timeout), cancelling the call's database, Redis, and storage
  work; the handler then answers `500 INTERNAL_ERROR`.
- **Auth.** `Authorization: Bearer <JWT>` (HS256, configured secret, issuer
  `sakukota-go`, unexpired; anything else rejected). The token carries user ID and
  role (`user`/`admin`). No valid token → `401 UNAUTHORIZED`; non-admin on an
  admin route → `403 FORBIDDEN`. Admin checks live in the router.
- **Route access** (the router test fails on any missing or extra route):
  - *Public:* `/docs`, `/openapi.json`, `/v1/auth/*`, `GET /places`,
    `GET /places/{id}`, `GET /places/{id}/reviews`, `GET /places/categories[/{id}]`,
    `GET /subdistricts[/{id}]`.
  - *Signed in:* `POST /places/{id}/reviews`, `DELETE /places/{id}/reviews/mine`,
    the caller's review media routes under `/places/{id}/reviews/mine/medias`,
    `POST /suggestions`, `GET /suggestions/mine`, `POST /places/{id}/reports`,
    `POST /places/{id}/reviews/{reviewId}/reports`.
  - *Admin:* place and sub-district create/update/delete and media routes,
    `PUT /subdistricts/{id}/boundary`, `DELETE /places/{id}/reviews/{reviewId}`,
    `GET /suggestions[/{id}]`, approve, reject, `GET /reports[/{id}]`,
    `POST /reports/{id}/remove`, `POST /reports/{id}/dismiss`.
- **Security headers.** Every response, errors and 404s included, carries
  `Cache-Control: no-store`, `Content-Security-Policy: default-src 'none';
  frame-ancestors 'none'`, `Cross-Origin-Resource-Policy: same-origin`,
  `Permissions-Policy: geolocation=(), camera=(), microphone=()`,
  `Referrer-Policy: no-referrer`, `Strict-Transport-Security:
  max-age=31536000`, `X-Content-Type-Options: nosniff`, and
  `X-Frame-Options: DENY`. The `/docs` HTML page alone relaxes the CSP to
  `frame-ancestors 'none'` so its scripts load.
- **Privacy.** Emails appear only in the caller's own user object from register,
  login, and refresh. Nested authors are `{id, name}`. No password material.

## 6. Behavior by module

### 6.1 Auth (`internal/module/auth`)

Routes under `/v1/auth`. Config: `JWTSecret`, `JWTExpiry`, `RefreshTokenExpiry`,
`CookieSecure`.

- **Register.** `name` (1–100), `email` (trimmed, lower-cased, ≤255, one `@`
  with text on both sides, no spaces), `password` (≥8 code points, argon2id).
  Invalid → `VALIDATION_ERROR`; duplicate → `409 EMAIL_TAKEN`. Returns `201` with
  `{id, name, email, role, created_at, updated_at}`.
- **Login.** Unknown email or wrong password → `401 INVALID_CREDENTIALS`, no
  cookie. Storage errors and corrupt hashes → 500. Success: `200 {access_token,
  user}`; the refresh token is only in the `refresh_token` cookie (HttpOnly,
  `Path=/v1/auth`, `SameSite=Strict`, `Secure=CookieSecure`).
- **Refresh.** No cookie, unknown token, or deleted user → `401
  INVALID_REFRESH_TOKEN` and the cookie is cleared. Tokens rotate; reuse fails.
  Storage errors → 500.
- **Logout.** Always clears the cookie; `204`, or 500 if revocation fails.

### 6.2 Users (`internal/module/users`)

Service only (`Create`, `FetchByEmail`, `Fetch`); passes `domain.ErrEmailTaken`
and `domain.ErrUserNotFound` through. Role defaults to `user`.

### 6.3 Places and categories (`internal/module/places`)

**Response:** `id, sub_district_id, name, address, location, operating_hours,
estimated_price_range, short_description, attributes, rating_average,
rating_count, medias, category{id,name,slug,code,label,color}`. Missing values
are `null`, attributes default to `{}`, `rating_average` is `null` without
ratings, `medias` is `[]` without images.

- **Fields.** Name required (may repeat). Location optional, in the Indonesia box.
- **Attributes** are validated against the category's fields: types `text`,
  `number`, `boolean`, `select` (one of `options`), `range` (`{min, max}`,
  `min ≤ max`), `date` (`YYYY-MM-DD`). Required fields must be present and
  non-null (text non-blank); unknown keys are rejected; `null` optional fields
  are dropped; non-objects are rejected.
- **Create/Update.** Every validation failure, including an unknown category or
  sub-district, is `400 VALIDATION_ERROR`. Update validates before looking up the
  place (unknown → `404 PLACE_NOT_FOUND`) and keeps the rating and medias.
- **Get.** Unknown → `404 PLACE_NOT_FOUND`.
- **Delete.** `204`, then `404`. Medias, including its reviews' medias, are
  deleted and their keys queued in `STORAGE_DELETIONS` in the same statement;
  reviews are deleted; approved
  suggestions keep their status with `place_id = null`.
- **List (`GET /places`)** returns `{places, total}`; `total` counts all matches
  before paging. Filters combine with AND; empty parameters count as absent.
  - `sub_district_id`: unknown → `404 SUB_DISTRICT_NOT_FOUND`.
  - `category_id`: comma list, matches any; unknown → `404 CATEGORY_NOT_FOUND`,
    malformed → `INVALID_CATEGORY_ID`.
  - `q`: case-insensitive literal substring of name, address, description, or
    category name (not attributes).
  - `limit`: positive integer (`INVALID_LIMIT`); absent or >100 becomes 100.
  - `offset`: non-negative integer (`INVALID_OFFSET`).
  - `bbox=west,south,east,north`: valid lon/lat with `west ≤ east`,
    `south ≤ north` (`INVALID_BBOX`); edges inclusive; places without a location
    are excluded.
  - **Order.** Without `bbox`, by id. With `bbox`, **spread over the box**: the
    box is split into about `limit` power-of-two cells anchored at 0,0; each
    cell rotates through its categories (starting offset varies by cell), nearest
    to the cell centre first; round *n* takes the *n*-th place of every cell,
    nearest the box centre first, ties by id. The order depends on `limit`, so
    pages must share one.
  - **Cache.** Lists with a `sub_district_id` are served from a Redis index and
    must match the uncached result exactly. Others are never cached.
- **Medias** (max 10, positions `1..n`):
  - `POST /places/{id}/medias/upload-url {content_type}` presigns an upload under
    `places/{id}` (unknown place → 404).
  - `POST /places/{id}/medias {upload_key}` promotes and appends it, returning
    the place. Concurrent adds never exceed 10. An 11th → `409
    MEDIA_LIMIT_REACHED`. If storing fails, the promoted object is deleted.
  - `DELETE /places/{id}/medias/{mediaId}` → `204`, queues the key, shifts later
    medias down. Not this place's / unknown → `404 MEDIA_NOT_FOUND`.
  - Upload errors → `400 VALIDATION_ERROR`. No reordering.
- **Categories.** `GET /places/categories[/{id}]` return
  `{id,name,slug,code,label,color,fields[]}`, ordered by id, fields by
  `sort_order`. Each field: `{id, category_id, key, label, data_type, required,
  options|null, sort_order}`. Unknown → `404 CATEGORY_NOT_FOUND`.

  The 16 categories come from migrations (untested): Warung, Minimarket, Air &
  Gas (chili); Penjahit, Fotokopi (plum); Klinik, Rumah Sakit, Puskesmas, Pos
  Polisi (cat); Laundry, Bank & ATM, Halte/Stasiun, Layanan Publik (sign);
  Apotek, Masjid, Ibadah Lain (board).

### 6.4 Reviews (`internal/module/reviews`)

- `GET /places/{id}/reviews` (public): newest first, ties by higher id; unknown
  place → 404.
- `POST /places/{id}/reviews {rating, comment?}` (signed in): `rating` integer
  1–5 (wrong type → `INVALID_BODY`, missing/out of range → `VALIDATION_ERROR`).
  One review per user per place (`409 ALREADY_REVIEWED`, holds under
  concurrency). The review is saved and its rating added to the place's
  `rating_count`/`rating_sum` through the place service in one transaction: a
  rejected review adds no rating, a failed rating discards the review, and
  concurrent reviews lose no rating.
- **Medias** of the caller's own review of the place (signed in; same rules
  as place medias: max 10, positions `1..n`):
  - `POST /places/{id}/reviews/mine/medias/upload-url {content_type}` presigns
    an upload under `reviews/{placeId}/{userId}`.
  - `POST /places/{id}/reviews/mine/medias {upload_key}` promotes and appends
    it, returning the review. Concurrent adds never exceed 10; an 11th → `409
    MEDIA_LIMIT_REACHED`. If storing fails, the promoted object is deleted,
    even if the request was cancelled.
  - `DELETE /places/{id}/reviews/mine/medias/{mediaId}` → `204`, queues the
    key, shifts later medias down. Another user's / unknown → `404
    MEDIA_NOT_FOUND`.
  - No review by the caller → `404 REVIEW_NOT_FOUND` (presign and add, before
    anything is presigned or promoted). Upload errors → `400
    VALIDATION_ERROR`.
- **Delete (admin) `DELETE /places/{id}/reviews/{reviewId}`** → `204`. The
  review is deleted, its medias deleted and their keys queued, and its rating
  taken off the place's totals through the place service, all in one
  transaction: a failed rating change keeps the review. Concurrent deletes take
  the rating off once. Bad ids → `INVALID_PLACE_ID` / `INVALID_REVIEW_ID`;
  unknown review or one of another place → `404 REVIEW_NOT_FOUND`. The author
  may then review the place again.
- **Delete own (signed in) `DELETE /places/{id}/reviews/mine`** → `204`, with
  the same effects as an admin delete. No review by the caller → `404
  REVIEW_NOT_FOUND`.
- Response: `{id, place_id, rating, comment, created_at, user{id,name},
  medias}`; `medias` is `[]` without any, and lists include each review's
  medias in position order. No edit.

### 6.5 Sub-districts (`internal/module/subdistricts`)

**Response:** `id, name, slug, flood_history, safety_crime_information,
estimated_daily_spending, medias, boundary` (GeoJSON MultiPolygon or `null`).

- **Validation.** Name and slug 1–100; `estimated_daily_spending` 0 to
  99,999,999.99.
- **Uniqueness.** Name case-insensitive, slug case-sensitive; conflict → `409
  SUB_DISTRICT_EXISTS` (holds under concurrency).
- **List** by id. **Update** is a PUT that keeps medias and boundary; create and
  update ignore `boundary`.
- **Delete.** Referenced by places → `409 SUB_DISTRICT_IN_USE`, nothing changes.
  Otherwise its suggestions are deleted and its media keys queued.
- **Medias.** Same rules as places under `/subdistricts/{id}/medias`.
- **Boundary (`PUT /subdistricts/{id}/boundary`, admin).** Body is a GeoJSON
  Polygon or MultiPolygon (stored as MultiPolygon), replacing the old one and
  returning the sub-district. Malformed GeoJSON → `INVALID_BODY`; a vertex
  outside Indonesia or a self-crossing shape → `VALIDATION_ERROR`; unknown → 404.
  Holes and multiple polygons are fine. A boundary cannot be cleared.

### 6.6 Suggestions (`internal/module/suggestions`)

- **Submit (`POST /suggestions`).** Place fields plus `note`. **Location is
  required.** Attributes must be an object but are not checked against the
  category until approval. Status is always `pending`. Unknown sub-district or
  category → `VALIDATION_ERROR`. Never creates a place.
- **Pending limit.** 5 per user → `429 TOO_MANY_PENDING_SUGGESTIONS` (holds
  under concurrency).
- **Response:** place fields plus `note, status, rejection_reason, place_id,
  reviewed_at, created_at, user{id,name}`. `reviewed_by` is never exposed.
- **Listing.** `/suggestions/mine`: newest first. `/suggestions?status=` (admin):
  oldest first; `pending|approved|rejected` or empty, else `400 INVALID_STATUS`.
  `/suggestions/{id}` (admin): unknown → `404 SUGGESTION_NOT_FOUND`.
- **Approve (admin).** Body is the final place (same shape as `POST /places`).
  Not pending → `409 ALREADY_REVIEWED`. The place is created through the place
  service (validation errors leave it pending) and the suggestion is marked
  approved in the same transaction; if marking fails, the place is rolled back
  and never visible. Concurrent approvals create exactly one place. The
  original submission is kept.
- **Reject (admin) `{reason?}`.** Not pending → 409. The author sees the reason.
- Reviewed suggestions are immutable. Deleting a user or sub-district deletes
  their suggestions.

### 6.7 Reports (`internal/module/reports`)

- **Report (signed in).** `POST /places/{id}/reports` reports the place;
  `POST /places/{id}/reviews/{reviewId}/reports` reports the place's review.
  Body `{reason, note?}`: `reason` is `spam`, `offensive`, `fake`, or `other`
  (wrong JSON type → `INVALID_BODY`; missing or other value → `VALIDATION_ERROR`);
  `note` is optional, ≤1000. Status is always `pending`. Returns `201`.
  - Unknown place → `404 PLACE_NOT_FOUND`; unknown review, or one of another
    place → `404 REVIEW_NOT_FOUND`. Bad ids → `INVALID_PLACE_ID` /
    `INVALID_REVIEW_ID`.
  - One pending report per user and content → `409 ALREADY_REPORTED`; once
    resolved, the user may report it again. Another user may report the same
    content. Holds under concurrency.
  - **Pending limit.** 5 per user → `429 TOO_MANY_PENDING_REPORTS` (holds under
    concurrency); a resolved report frees a slot.
- **Response:** `{id, place_id, review_id|null, reason, note, status,
  resolved_at, created_at, user{id,name}}`. `resolved_by` is never exposed.
- **Show (admin) `GET /reports/{id}`:** the report plus its content as it is
  now: `place {id, name, address, short_description, category, medias}` (the
  place, or the reviewed place) and, for a review report, `review {id, rating,
  comment, created_at, user{id,name}, medias}`. Deleted content is `null`;
  `review` is `null` for a place report. Read through the place and review
  services. Unknown → `404 REPORT_NOT_FOUND`; bad id → `INVALID_REPORT_ID`.
- **List (admin) `GET /reports?status=`:** oldest first, without content;
  `pending|removed|dismissed` or empty, else `400 INVALID_STATUS`.
- **Remove (admin) `POST /reports/{id}/remove`** → `200` with the report. In one
  transaction it marks the report `removed`, deletes the place (as place delete)
  or the review (as review delete, taking its rating off), and marks `removed`
  every other pending report of that content; removing a place also resolves
  the pending reports of its reviews. A failed delete keeps the report pending.
  Content already deleted another way still resolves the report. Concurrent
  removes delete once; the others get `409`.
- **Dismiss (admin) `POST /reports/{id}/dismiss`** → `200`; marks only that
  report `dismissed` and keeps the content.
- Both: not pending → `409 ALREADY_RESOLVED`; unknown → `404 REPORT_NOT_FOUND`;
  bad id → `INVALID_REPORT_ID`. Resolved reports are immutable.
- **History.** Reports keep only the ids of their content (no foreign keys), so
  they outlive the place or review. Deleting the reporter deletes their
  reports.

### 6.8 Media storage (`internal/storage`)

- **Types:** `image/jpeg`, `image/png`, `image/webp` (≤5 MB), `video/mp4`,
  `video/webm` (≤50 MB). Anything else, or parameters → `ErrInvalidMediaType`.
- **Presign** returns a unique key `uploads/{prefix}/{uuid}.{ext}` and a presigned
  PUT URL on the public endpoint, valid for `UploadTTL`, with the content type
  signed.
- **Promote** checks the key format (`ErrInvalidUploadKey`), existence
  (`ErrMediaNotUploaded`), size, and type/extension match; rejected uploads are
  deleted. Otherwise it copies to `{prefix}/{name}`, deletes the upload, and
  returns a `Media`. All errors map to `400 VALIDATION_ERROR`.
- Media responses are `{id, url, content_type, size, position}`; `url` is the
  public base plus key.

### 6.9 Storage cleanup (`internal/module/storagecleanup`)

`STORAGE_DELETIONS` queues keys of deleted medias. `cmd/cleanup` runs one batch
of up to 100 due keys: success removes the row; failure records the error and
retries in 1 hour without failing the run. Overlapping runs may read the same
rows, which is harmless because deletes are idempotent.

### 6.10 Code structure

Entities, shared rules and every sentinel error live in `internal/domain`.
Modules never import each other: a module that needs another (auth → users,
suggestions → places, reviews → places, reports → places and reviews) declares the interface it consumes, and
`internal/router` wires the concrete services. Each handler declares the
`Service` interface it consumes, and `NewService` returns it, backed by the
unexported `service` struct (`places.Service` also carries `AddRating` and
`RemoveRating` for reviews, `reviews.Service` carries `FetchByID` for
reports). Users and storage cleanup have no handler, so their `NewService`
returns `*service`. Services own transactions that
span repositories or services through `database.Transactor.WithinTx`, which
carries the transaction in `ctx`; repositories run queries on
`database.Conn(ctx, db)` to join it (place delete included), and a nested
`WithinTx` joins the outer transaction. Each module's handler maps `domain` errors to codes;
only the five cross-cutting codes live in `internal/pkg/httpx`. Tests use
`testing` with testify and gomock; mocks are generated by `mockgen` into each
module's `mock/` package (`go generate ./...`).

### 6.11 Telemetry (`internal/middleware`, `internal/database`)

`WithTelemetry` traces every request and records OpenTelemetry HTTP server
metrics. Metrics are labelled and spans named by the matched chi route pattern
(`GET /v1/places/{id}`), never the raw path; unmatched requests keep the
method-only span name. `server.address` is fixed, not taken from the client's
`Host`. PostgreSQL queries (otelpgx) and Redis commands (redisotel) become
child spans and report timings and pool usage.

`WithRequestLog` logs one slog line per request (method, matched route pattern
or empty, status — 200 when nothing was written — bytes, duration) with the
request context; it never logs the raw path, query string, or client address.

All are opt-in in `cmd/api`: `METRICS_ADDR` serves `/metrics` (with Go
runtime and `database/sql` pool metrics) on a separate listener,
`OTEL_EXPORTER_OTLP_ENDPOINT` exports traces over OTLP/HTTP, and
`OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` also sends slog records over OTLP with
their trace and span IDs (stderr output stays). Compose wires Prometheus,
Tempo (local storage inside its container), Loki (OTLP ingestion), and
Grafana with all three sources, span-to-logs and log-to-trace links, and the
"SakuKota API" dashboard provisioned.
`cmd/cleanup` honours the same two endpoints: each run is a `storage cleanup`
trace, marked as an error when the run fails; its log lines (a `storage cleanup
run` line with the `due` count on success, the error on failure, including
config and connection errors) link to it. Both are flushed before the command
exits, with status 1 on failure.

## 7. Error codes

| Code | Status | When |
| ---- | ------ | ---- |
| `INVALID_BODY` | 400 | Undecodable or wrongly typed body |
| `VALIDATION_ERROR` | 400 | Business-rule violations, unknown referenced sub-district/category in writes, upload errors |
| `INVALID_*_ID` (place, sub-district, category, suggestion, media, review, report) | 400 | Bad path id or `GET /places` filter |
| `INVALID_LIMIT` / `INVALID_OFFSET` / `INVALID_BBOX` | 400 | `GET /places` |
| `INVALID_STATUS` | 400 | `GET /suggestions`, `GET /reports` |
| `UNAUTHORIZED` | 401 | Missing or invalid access token |
| `INVALID_CREDENTIALS` / `INVALID_REFRESH_TOKEN` | 401 | Auth |
| `FORBIDDEN` | 403 | Non-admin on an admin route |
| `*_NOT_FOUND` (place, sub-district, category, suggestion, media, review, report) | 404 | Unknown resource, or unknown `GET /places` filter id |
| `EMAIL_TAKEN` / `SUB_DISTRICT_EXISTS` / `SUB_DISTRICT_IN_USE` / `ALREADY_REVIEWED` / `MEDIA_LIMIT_REACHED` / `ALREADY_REPORTED` / `ALREADY_RESOLVED` | 409 | Conflicts |
| `TOO_MANY_PENDING_SUGGESTIONS` / `TOO_MANY_PENDING_REPORTS` | 429 | Pending limits |
| `INTERNAL_ERROR` | 500 | Anything unexpected |

An unknown sub-district or category is **404** as a `GET /places` filter but
**400 VALIDATION_ERROR** in a write body.

## 8. User stories

| Story | Backend support |
| ----- | --------------- |
| US-01 Browse pins in viewport | `GET /places?bbox=…`, ≤100 spread places, `total` |
| US-02 Pin details | Place response, `null` for missing values |
| US-03 Kecamatan dropdown | `GET /subdistricts` |
| US-04 Kecamatan info | Flood, safety, spending, boundary |
| US-05 Category filter | `category_id=1,2` |
| US-06 Keyword search | `q` |
| US-07 Filters synced with map | Filters and `bbox` AND together; `total` reflects them |

## 9. Design docs vs. tests

- A place has **one** `category_id` with typed `attributes` (not the ERD's
  many-to-many).
- Kecamatan info lives in `SUB_DISTRICTS`; the static map image became a PostGIS
  `boundary`.
- Coordinates are PostGIS points exposed as GeoJSON, not lat/lng columns.
- Routes are under `/v1` with UPPER_SNAKE codes; categories are at
  `/v1/places/categories`.
- Users, reviews, admin, and suggestions exist despite the ERD's read-only scope.
- Up/down votes became 1–5 star reviews.

## 10. Gaps and open questions

- **Untested code:** middleware beyond the router test, `config`, most of
  `httpx`, `postgis`, `database` (except `Transactor`), `cmd/*`, seed data, repository SQL, and the
  migration-defined categories.
- **Specified but untested:**
  - 1 MiB body cap (`INVALID_BODY`); server timeouts (10s/30s/30s/120s).
  - Startup fails on invalid env values or a `JWT_SECRET` under 32 bytes.
  - Migration indexes and `CHECK`s (role, field data type, suggestion limits).
  - Orphan cleanup ignores request cancellation.
  - No CORS headers: the web app uses a same-origin proxy.
  - Refresh tokens live in Redis (`GETDEL`, Redis 6.2+); the API exits if Redis
    is unreachable at startup.
  - The places cache lives 30 s with no invalidation, so lists can be 30 s
    stale; Redis failure should fall back to PostgreSQL.
- **Known gaps:** a media added while its owner is deleted can be orphaned;
  deleting a user cascades to their reviews but leaves the reviews' `MEDIAS`
  rows and files unqueued;
  medias cannot be reordered; `MEDIAS` has no owner type; a report shows its
  content live, so once removed it cannot show what the content said, and
  content deleted outside a report leaves its reports pending; bucket objects from the
  old static maps were not queued (the migration was edited in place, so
  databases must be rebuilt).
- **Unspecified:** `UploadTTL`, production token lifetime, cleanup schedule,
  review edit, a current-user endpoint, listing users, a reporter's view of
  their own reports.
- **Performance:** the `bbox` spread ranks every place in the box (~4 ms on
  1,092 seeded places; untested at 50,000).
- **Web (`web/`):** untested. The map sends
  a `limit` sized to its screen area (20–100).
- **`scripts/k6/get-places.js`** still reads the removed `X-Cache` header.

## Maintaining this file

- Update only after VERIFICATION passes, and only for verified behavior.
- Every stated behavior must trace to a test or the user's approved spec, never
  to production code. Remove resolved items from §10.
- Bump "Last updated" and add a one-line change log entry.

### Change log

| Date | Change |
| ---- | ------ |
| 2026-11-01 | Initial draft from the planning docs and tests; null bodies, length limits, trimming, cleanup queue |
| 2026-11-05 | 16 categories with themed colors |
| 2026-11-09 | `PATCH` → `PUT`; route access and JWT rules tested; refresh tokens in Redis |
| 2026-11-11 | Shared `DecodeJSON`; `role` in user objects; no API CORS |
| 2026-11-13 | Multiple medias (images and video) per place and sub-district; plural module packages |
| 2026-11-15 | `GET /places` limit cap lowered |
| 2026-11-16 | `X-Cache` header removed; auth simplified (no max password length) |
| 2026-11-17 | Cleanup reads instead of claiming, fixed 1 h retry; storage renames; real seed data; places ordered by distance; cap 100 |
| 2026-11-22 | Static map replaced by GeoJSON `boundary`; `bbox` pages spread over the box; seed and web updated |
| 2026-11-22 | Documentation simplified; no contract change |
| 2026-11-23 | Shared `internal/domain`; no module-to-module imports; consumer-side interfaces; module-owned error codes; testify and gomock tests with generated mocks; no contract change |
| 2026-11-23 | Reviews add ratings through the place service in a service-owned transaction (`database.Transactor`); no contract change |
| 2026-11-23 | Handler-owned exported `Service` interfaces returned by `NewService`; unexported `service` structs; no contract change |
| 2026-11-23 | OpenTelemetry HTTP metrics by route on an opt-in `METRICS_ADDR` listener; Prometheus and Grafana in compose; no contract change |
| 2026-11-23 | Tracing (HTTP, PostgreSQL, Redis) to Jaeger, pool metrics, provisioned Grafana dashboard; no contract change |
| 2026-11-23 | Traces go to Tempo instead of Jaeger; no contract change |
| 2026-11-23 | Each `cmd/cleanup` run is traced as `storage cleanup`; no contract change |
| 2026-11-23 | slog request log replaces chi's `Logger`; logs exported to Loki with trace IDs, linked with Tempo in Grafana; no contract change |
| 2026-11-23 | `cmd/cleanup` logs to Loki with a per-run summary; request log records the route pattern instead of the raw path; no contract change |
| 2026-11-23 | Review medias: presign/add/delete routes under `/places/{id}/reviews/mine/medias`, `medias` in review responses; place delete queues review medias |
| 2026-11-23 | Web review form picks up to 10 files and adds them through the review media routes once the review is saved; no contract change |
| 2026-11-23 | Web review images open full size in a modal on click; no contract change |
| 2026-11-24 | Admin review deletion `DELETE /places/{id}/reviews/{reviewId}` (rating removed, medias queued); web shows a delete button with confirmation to admins |
| 2026-11-24 | Authors delete their own review via `DELETE /places/{id}/reviews/mine`; web shows them the delete button too |
| 2026-11-24 | Abuse reports of places and reviews (`REPORTS`, reasons, pending limit 5); admins list, remove content, or dismiss; place delete joins the caller's transaction; web report drawer and `/admin/reports` |
| 2026-11-24 | `GET /reports/{id}` returns the reported place and review; web `/admin/reports/$reportId` shows them and holds remove and dismiss |
| 2026-11-25 | Docs only: unpublished planning-doc paths generalized for public release; README disclaimer |
| 2026-11-25 | Project renamed to SakuKota (module `sakukota-go`, database `sakukotago`, bucket `sakukota-images`, demo password `sakukota123`) |
| 2026-11-25 | README rewritten to the portfolio template; earlier README versions removed from history |
| 2026-11-25 | Licensed under AGPLv3; product, backend, and frontend docs dropped (and removed from history), leaving this file |
| 2026-11-25 | `.env.example` lists every variable with working local values (32-byte `JWT_SECRET`); compose bucket CORS follows `WEB_PORT`; no contract change |
| 2026-11-25 | Web dev server port (`DEV_PORT`, strict) and API proxy target (`API_PROXY_TARGET`, set by `make dev` from `PORT`) configurable; compose api fixed to container port 8080; no contract change |
| 2026-11-25 | Web `/admin/subdistricts`: list, create (slug follows the name until edited), edit, delete, images, and a GeoJSON boundary picked in the browser, previewed, and sent to `PUT /subdistricts/{id}/boundary`; no contract change |
| 2026-11-25 | Security headers (CSP, HSTS, nosniff, frame denial, no-store, CORP, referrer policy) on every response; `/docs` keeps only `frame-ancestors` |
| 2026-11-25 | `Permissions-Policy` (no geolocation, camera, microphone) added to the security headers |
| 2026-11-25 | Request contexts expire after 25 s (`WithTimeout`); a timed-out call answers 500 |
