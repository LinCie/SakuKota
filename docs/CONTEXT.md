# Rantau — Project Context

> **Living document.** This file states what the project is *for* and what the
> backend is *specified to do*. Update it only after a VERIFICATION phase
> completes (see [Maintaining this file](#maintaining-this-file)).
>
> - Last updated: 2026-11-05. Updated after a verification pass of the category expansion (see the change log).
> - Status: verified. The place tests pass, the migration and seed were applied to a clean database, and the web app type-checks, lints, and builds.

## Contents

1. [Why the project exists](#1-why-the-project-exists)
2. [Glossary](#2-glossary)
3. [Sources of truth](#3-sources-of-truth)
4. [Scope: planned vs. delivered](#4-scope-planned-vs-delivered)
5. [Cross-cutting API contract](#5-cross-cutting-api-contract)
6. [Behavior by module](#6-behavior-by-module)
7. [Error code catalogue](#7-error-code-catalogue)
8. [User stories → backend support](#8-user-stories--backend-support)
9. [Where the design docs and the tests disagree](#9-where-the-design-docs-and-the-tests-disagree)
10. [Gaps and open questions](#10-gaps-and-open-questions)
11. [Maintaining this file](#maintaining-this-file)

---

## 1. Why the project exists

The source docs (not published) call the project **Perantau Helper**.
In this repo it is called **Rantau** / `rantau-go`.

- **Problem.** People who move to Jakarta for work (*perantau*) struggle to find
  cheap everyday essentials: warung, tailors, local clinics, laundries,
  pharmacies, and the places a newcomer soon needs (mosques, police posts,
  hospitals, puskesmas, banks and ATMs, minimarkets, transit stops, public
  service offices, water and gas refills, print shops). Mainstream maps either miss these places or bury them. Newcomers
  also know little about each neighbourhood: flood risk, safety, and the cost
  of daily life. Today they rely on trial and error or word of mouth.
- **Target users.** Fresh graduates, trainees, and young professionals moving to
  Jakarta.
- **Persona.** Dimas, 23, a corporate trainee who moved from Surabaya to Setiabudi.
  He mostly uses the app on his phone after work. He wants a cheap dinner and a
  nearby laundry, and he wants to check his kecamatan's flood, safety, and
  spending information.
- **Unique value.** A hyper-local "survival map" that combines curated daily
  essentials with practical safety and cost information for each neighbourhood.
- **Hard non-goal.** Kost or boarding-house listings ("Won't Have"). The app
  should not compete with property apps.
- **Quality stance.** Data quality beats volume. The docs favour admin-verified
  pins backed by community signals over unverified clutter, and placeholders
  over broken layouts when data is missing.

## 2. Glossary

| Term | Meaning |
| ---- | ------- |
| Perantau | A person who has moved away from home for work or study |
| Kecamatan / sub-district | Jakarta administrative district. The code and API call it `subdistrict` / `sub_district` |
| Place / pin | A curated essential spot shown on the map |
| Category | Place type, e.g. Warung. Defined in migrations, never created through the API. Each category has `code`, `label`, and `color` (a palette key the frontend maps to styles) plus custom **fields** |
| Attributes | A place's values for its category's custom fields (a JSON object) |
| Suggestion | A place proposed by a signed-in user that waits for admin review |
| Review | A user's 1–5 star rating of a place, with an optional comment |

## 3. Sources of truth

1. **Tests (`*_test.go`) are the executable specification.** They decide what
   the backend must and may do. Production code never counts as evidence of
   intent (see the root `AGENTS.md`).
2. **This file** summarizes the intent behind the docs and the current tested
   contract. When this file and a test disagree, the test wins until PLANNING
   resolves the disagreement with the user.
3. **Design docs** (not published): `ideation_scope.md`,
   `user_stories.md`, `erd.md`, `wireframe/`, and `weekly_report/week_2..11.md`.
   They explain the original *why*, but some parts are out of date (see §9).

## 4. Scope: planned vs. delivered

MoSCoW scope agreed with the mentor in Week 3:

| Tier | Feature | Backend status (per tests) |
| ---- | ------- | -------------------------- |
| Must | Interactive survival map and essential pins | Delivered: places API with filters, bbox, limit, total |
| Must | Kecamatan dropdown and info modal (flood, safety/crime, daily spending, static map image) | Delivered: sub-districts API with uploaded images |
| Must | Category filter and search | Delivered: `category_id` (any of several), `q` |
| Should | Basic admin verification panel | Backend delivered: admin review of suggestions; create/update/delete for places and sub-districts (which roles may call them is untested, see §10) |
| Could | Place suggestion form | Delivered: suggestions with a pending limit |
| Could | Upvote/downvote and review system | Delivered as **1–5 star reviews** with comments (not up/down votes) |
| Could | Kecamatan GeoJSON overlay | Not present |
| Won't | Kost listings | Not present (keep it that way) |

Also delivered, beyond the docs: authentication with JWT access tokens and
rotating refresh cookies, image uploads to S3-compatible storage (Garage), and
a durable cleanup queue for deleting stored objects.

Every user story in the docs is a Must Have. The Should and Could features have
**no user stories or acceptance criteria in the docs**. Their requirements exist
only as tests.

## 5. Cross-cutting API contract

These rules apply to every HTTP module unless §6 says otherwise.

- **Envelopes.** A success response is `{"data": ...}`. An error response is
  `{"code": "UPPER_SNAKE", "message": "<non-empty>"}` with
  `Content-Type: application/json`.
- **Status codes.** Create returns `201` with the created resource. Read,
  update, and actions return `200`. Delete returns `204` with an empty body.
- **500s never leak internals.** A response body never contains driver or SQL
  details (`pq:`, relation names, "database is closed"), and its code is
  always `INTERNAL_ERROR`.
- **Path IDs.** An ID must be a positive integer that fits in 64 bits.
  Non-numeric, fractional, zero, negative, or overflowing IDs return `400
  INVALID_<RESOURCE>_ID`, and the service is never called.
- **`INVALID_BODY` vs `VALIDATION_ERROR`.** A body that cannot be decoded
  returns `400 INVALID_BODY`. That covers an empty body, malformed JSON, a body
  that is not an object (array or `null`), wrong JSON types, a non-Point
  GeoJSON, or coordinates outside WGS84 (lat > 90, lon > 180). A body that
  decodes but breaks a business rule returns `400 VALIDATION_ERROR`. Requests
  with invalid bodies never reach the service. The rule applies to every JSON
  endpoint, including register, login, create, approve, reject, and review, and
  a `null` body counts as an invalid body.
- **Client-controlled fields are ignored.** The server ignores `id`,
  `created_at`, `user_id`, `status`, review fields, and any unknown keys sent
  by the client. The author always comes from the access token, and the
  resource ID always comes from the path.
- **PATCH semantics.** A field that is absent stays unchanged. A field sent as
  `null` clears a nullable field. A field sent with a value replaces it. Sending
  `null` for a required field (`name`, `slug`, `sub_district_id`,
  `category_id`) returns `INVALID_BODY`. An empty object `{}` changes nothing
  and returns 200. A failed update leaves stored data unchanged.
- **Text normalization.** Names and free-text inputs are trimmed of surrounding
  whitespace, including tabs and newlines. Inner whitespace and line breaks are
  kept. Length limits count Unicode **code points after trimming**, so `é`
  counts as 1 and a ZWJ family emoji counts as 5. Optional text on places,
  sub-districts, and suggestions is trimmed the same way. Empty or blank optional
  text is stored as absent (`null`).
- **Length limits.** Every text field with a column limit is checked against it
  using the same count: user name 100, user email 255, sub-district name and slug
  100, place name 150, place address 255, operating hours 255, estimated price
  range 100, suggestion name 150 (with the same address, hours, and price limits
  as places), suggestion note and rejection reason 1000, and review comment 1000.
  A value over its limit returns `400 VALIDATION_ERROR` and never reaches storage.
  Free text without a column limit (place short description, sub-district flood
  history and safety information) has no length check.
- **Locations.** A location is a GeoJSON `{"type":"Point","coordinates":[lon, lat]}`.
  A domain location must fall inside the **Indonesia box**: lon 95–141 and lat
  −11–6, edges included. `[0,0]` is rejected. Locations come back exactly as
  they were sent.
- **JSON fidelity.** Attribute JSON round-trips exactly: big integers keep every
  digit, `1.50` stays `1.50`, and quotes, emoji, and SQL-like text are kept
  verbatim.
- **Context.** Every service call receives the request's context. A cancelled
  request or an expired deadline must not produce a success response.
- **Auth.** Protected routes use `middleware.WithAuth`, which requires
  `Authorization: Bearer <JWT>`. The JWT carries the user ID and role (`user` or
  `admin`). A request without a token returns 401. A handler that needs a user
  but finds none in the context returns `401 UNAUTHORIZED`. Admin-only routes
  are guarded by the **API router**, not by the handlers.
- **Privacy.** No response exposes user emails. A nested author appears as
  `{"id", "name"}`. Password material never appears in any response.

## 6. Behavior by module

### 6.1 Auth (`internal/module/auth`)

The test router posts to `/v1/auth`, and the refresh cookie path is
`/v1/auth`. Configuration: `JWTSecret`, `JWTExpiry` (access token),
`RefreshTokenExpiry`, `CookieSecure`.

- **Register.** `name`, `email`, and `password` are required. The name is
  trimmed, must not be blank, and has at most 100 code points. The email is
  trimmed, lower-cased, has at most 255 code points, and must be a bare address
  such as `ana@example.com`; a display-name form such as `Ana <ana@example.com>`
  is rejected. The password needs at least 8 characters. The password is stored
  as an argon2id hash. Invalid input returns `400 VALIDATION_ERROR` and never
  reaches storage. A duplicate email returns `409 EMAIL_TAKEN`. Success returns
  `201` with the user, never the password.
- **Login.** The email is normalized before lookup. An unknown email and a wrong
  password both return `401 INVALID_CREDENTIALS`, and neither sets a cookie.
  Storage failures and corrupt stored hashes are **not** reported as bad
  credentials: they return 500. Success returns `200 {access_token, user}`.
  The refresh token travels **only** in the `refresh_token` cookie (HttpOnly,
  `Path=/v1/auth`, `SameSite=Strict`, `Secure=CookieSecure`,
  `MaxAge=RefreshTokenExpiry`).
- **Refresh.** Reads the cookie. Without a cookie it returns `401
  INVALID_REFRESH_TOKEN`, clears the cookie, and does not call the service.
  Refresh tokens
  **rotate**: the old token is consumed, and reusing it fails. An unknown token
  or a deleted user returns `INVALID_REFRESH_TOKEN` and clears the cookie.
  Storage failures return 500 and are not reported as an invalid token.
- **Logout.** Always returns `204` and clears the cookie. Without a cookie the
  service is not called. If revocation fails, the response is 500 but the
  cookie is still cleared. A revoked token can no longer refresh.

### 6.2 Users (`internal/module/users`)

This is a service only, with no HTTP handler. It offers `Create`, `GetByEmail`,
and `GetUser`, which pass repository results and errors (`ErrEmailTaken`,
`ErrNotFound`) through unchanged. Stored users carry a `Role`, which defaults
to `"user"`.

### 6.3 Places and categories (`internal/module/place`)

**Place response:** `id, sub_district_id, name, address, location,
operating_hours, estimated_price_range, short_description, attributes,
rating_average, rating_count, image_url, category{id,name,slug,code,label,color}`.
Missing optional values are `null` and attributes default to `{}`.
`rating_average` is `rating_sum / rating_count`, or `null` when the place has
no ratings. `image_url` is the storage public URL of the image key, or `null`.

- **Name.** Required, trimmed, at most 150 code points (`ErrInvalidName`,
  `ErrNameTooLong`). Names may repeat, even in the same district and
  regardless of case.
- **Location.** Optional for places. When given, it must lie in the Indonesia
  box (`ErrInvalidLocation`).
- **Optional text.** `address` (at most 255 code points), `operating_hours`
  (255), and `estimated_price_range` (100) are trimmed and checked with
  `ErrAddressTooLong`, `ErrOperatingHoursTooLong`, and
  `ErrEstimatedPriceRangeTooLong`. `short_description` is trimmed. Blank values
  are stored as `null`. Patches follow the same rules.
- **Attributes** are validated against the category's fields:
  - Field types are `text`, `number`, `boolean`, `select` (one of `options`),
    `range` (exactly `{min, max}`, both numbers, `min ≤ max`), and `date`
    (a real date in `YYYY-MM-DD` form, without a time).
  - Required fields must be present and non-null. Required text must not be
    blank.
  - Unknown keys are rejected. A `null` optional field is dropped.
  - Absent or `null` attributes are stored as `{}`. Anything other than an
    object is rejected.
  - Violations return `ErrInvalidAttributes`.
- **Create (`POST /places`).** The category is looked up first; an unknown
  category stops before storage. Storage may report an unknown sub-district or
  category. Every create validation failure returns `400 VALIDATION_ERROR`,
  including an unknown sub-district or category.
- **List (`GET /places`).** Returns `{places, total}`, where `total` counts all
  matches **before** `limit`. Places are ordered by ascending id, and `limit`
  keeps the lowest ids. Filters combine with AND, and an empty parameter counts
  as absent.
  - `sub_district_id`: a positive int64. An unknown id returns `404
    SUB_DISTRICT_NOT_FOUND`.
  - `category_id`: a comma list of positive int64s that matches any of them.
    Repeated ids add no duplicate rows. An unknown id returns `404
    CATEGORY_NOT_FOUND`. A malformed list returns `INVALID_CATEGORY_ID`.
  - `q`: trimmed, and blank means no search. Case-insensitive substring match
    on name, address, short description, and category name. `%` and `_` are
    literal. Attributes are **not** searched.
  - `limit`: a positive int64, otherwise `INVALID_LIMIT`.
  - `bbox=west,south,east,north`: four finite numbers with lon in ±180, lat in
    ±90, `west ≤ east`, and `south ≤ north`, otherwise `INVALID_BBOX`. Edges
    count as inside, and places without a location are excluded. A GiST index
    on `PLACES.location` is required.
- **Get (`GET /places/{id}`).** Returns `404 PLACE_NOT_FOUND` for an unknown
  place.
- **Patch (`PATCH /places/{id}`).**
  - Nullable fields are `address`, `location`, `operating_hours`,
    `estimated_price_range`, and `short_description`. A `null` attribute sends
    attributes back to `{}`.
  - Changing the category re-validates the existing attributes against the
    new category. New attributes sent in the same patch are validated against
    the new category.
  - Unknown category or sub-district returns `400 VALIDATION_ERROR`. Unknown
    place returns `404`.
  - Edits keep the rating and the image.
- **Delete (`DELETE /places/{id}`).** Returns `204`, and a second delete
  returns `404`. If the place had an image, the **repository queues its key in
  `STORAGE_DELETIONS`** and the service does not delete it directly. The
  place's reviews are deleted with it. Approved suggestions that point to the
  place keep their status, with `place_id = null`.
- **Image.**
  - `POST /places/{id}/image/upload-url {content_type}` presigns an upload
    under `places/{id}` and returns `{upload_key, upload_url, method, headers,
    expires_at}`. An unknown place returns 404 before anything is presigned.
  - `PUT /places/{id}/image {upload_key}` promotes the upload, then stores the
    key. The repository queues the previous key for deletion; setting the same
    key again queues nothing.
  - If storing the key fails after promotion, the promoted object is deleted
    (best effort, failures ignored) and the old image is kept.
  - Promotion and type errors return `400 VALIDATION_ERROR`.
- **Categories.** `GET /categories` and `GET /categories/{id}` return
  `{id,name,slug,code,label,color,fields[]}`. `fields` is always an array,
  sorted by `sort_order` 1..n. `GET /categories` lists categories ordered by
  id. Each field is `{id, category_id, key, label,
  data_type, required, options|null, sort_order}`. Invalid ids return
  `INVALID_CATEGORY_ID`, and unknown ids return `404 CATEGORY_NOT_FOUND`.

  The categories are defined by migrations, so the set below is not covered by
  the unit tests (see §10). `color` is grouped by theme, and the web app tells
  categories of one hue apart by their `code` and icon.

  | Slug | Label | Code | Color | Required fields |
  | ---- | ----- | ---- | ----- | --------------- |
  | `affordable-food-warung` | Warung | W | chili | `halal`, `serving_style` |
  | `minimarket` | Minimarket | T | chili | `store_type` |
  | `water-gas` | Air & Gas | G | chili | `service` |
  | `tailor` | Penjahit | J | plum | `specialty`, `turnaround_days` |
  | `print-copy` | Fotokopi | F | plum | none |
  | `local-clinic` | Klinik | K | cat | `bpjs_accepted` |
  | `hospital` | Rumah Sakit | R | cat | `igd_24h`, `bpjs_accepted`, `ownership` |
  | `puskesmas` | Puskesmas | S | cat | `open_24h` |
  | `police-post` | Pos Polisi | P | cat | `facility_type`, `open_24h` |
  | `laundry` | Laundry | L | sign | `service_type`, `turnaround_hours` |
  | `bank-atm` | Bank & ATM | B | sign | `banks` |
  | `transit-stop` | Halte/Stasiun | H | sign | `mode` |
  | `public-service` | Layanan Publik | U | sign | `office_type` |
  | `pharmacy` | Apotek | A | board | `accepts_prescriptions` |
  | `mosque` | Masjid | M | board | `facility_type`, `friday_prayer` |
  | `worship-other` | Ibadah Lain | I | board | `religion` |

### 6.4 Reviews (`internal/module/review`)

- **`GET /places/{id}/reviews`** is public. It lists newest first, and on a
  tie the higher id comes first. An unknown place returns `404
  PLACE_NOT_FOUND`. A place without reviews returns `[]`.
- **`POST /places/{id}/reviews`** requires auth. The body is `{rating, comment?}`.
  - The place comes from the path and the author from the token.
  - `rating` must be an integer from 1 to 5. A string or fractional rating
    returns `INVALID_BODY`. A missing or out-of-range rating returns
    `VALIDATION_ERROR`.
  - The comment is trimmed, a blank comment is stored as absent, and the limit
    is 1000 code points.
  - **One review per user per place**, so a second review returns `409
    ALREADY_REVIEWED`. Under concurrent double submits, exactly one review is
    created.
  - Each review atomically updates the place's `rating_count` and
    `rating_sum`. The totals always match the stored reviews, including under
    concurrency.
- **Review response:** `{id, place_id, rating, comment, created_at (RFC 3339),
  user{id,name}}`.
- There are no edit or delete endpoints for reviews.

### 6.5 Sub-districts / kecamatan (`internal/module/subdistrict`)

**Response:** `id, name, slug, flood_history, safety_crime_information,
estimated_daily_spending, image_url, static_map_image_url`. Both URLs are
derived from stored image keys. `static_map_image_url` is **not** writable: the
server ignores it in create and patch bodies.

- **Validation.** The name is trimmed and must have 1–100 code points;
  violations return `ErrInvalidName`, which covers both blank and too long. The
  slug is trimmed and must have 1–100 code points (`ErrInvalidSlug`).
  `estimated_daily_spending` must be between 0 and 99,999,999.99, the range of
  `DECIMAL(10,2)` (`ErrInvalidEstimatedDailySpending`).
  `flood_history` and `safety_crime_information` are trimmed, and blank values
  are stored as `null`.
- **Uniqueness.** Names are unique **case-insensitively**, compared after
  trimming. Slugs are unique **case-sensitively**, so `MENTENG` is allowed next
  to `menteng`. A conflict returns `409 SUB_DISTRICT_EXISTS`, and the check
  holds under concurrent creates. Renaming a sub-district to its own name, or
  changing only the case, is allowed. Deleting a sub-district frees its name
  and slug.
- **List.** Ordered by id.
- **Patch.** `flood_history`, `safety_crime_information`, and
  `estimated_daily_spending` are nullable. Images are kept.
- **Delete.**
  - A sub-district that places still reference returns `409
    SUB_DISTRICT_IN_USE` and is not deleted.
  - Suggestions do **not** block deletion; they are deleted with the
    sub-district.
  - The repository **queues the deleted images in `STORAGE_DELETIONS`** in the
    same statement, and the service deletes nothing directly. The cleanup worker
    removes the objects.
- **Images.** There are two slots: `image` (stored under
  `subdistricts/{id}/image`) and `static-map` (stored under
  `subdistricts/{id}/static-map`). Each slot has
  `POST /subdistricts/{id}/<slot>/upload-url` and `PUT /subdistricts/{id}/<slot>`.
  Setting an image promotes the upload and stores the key in the same statement
  that queues the previous key in `STORAGE_DELETIONS`; setting the same key again
  queues nothing. The other slot is never touched. If
  storing the key fails, the promoted object is deleted.

### 6.6 Suggestions (`internal/module/suggestion`)

Any signed-in user can submit suggestions and list their own. Admins can list,
read, approve, and reject them. The admin role check happens in the API router.

- **Submit (`POST /suggestions`).** The body has the place fields plus `note`.
  - **Location is required** (unlike places) and must lie in the Indonesia box.
    Missing or out-of-range locations return `ErrInvalidLocation`.
  - The name is trimmed and limited to 150 code points.
  - The optional texts (`address`, `operating_hours`, `estimated_price_range`,
    `short_description`, `note`) are trimmed, and blank values are stored as
    absent. The note is limited to 1000 code points. `address` is limited to 255,
    `operating_hours` to 255, and `estimated_price_range` to 100.
  - Attributes must be a JSON object (absent or `null` become `{}`). They are
    **not** checked against the category until approval, so unknown keys and
    missing required fields are accepted at submit time.
  - The status is always `pending` and client-sent review fields are ignored.
  - An unknown sub-district or category returns `400 VALIDATION_ERROR`.
  - Submitting never creates a place.
- **Pending limit.** `MaxPendingPerUser = 5` per user. Exceeding it returns
  `429 TOO_MANY_PENDING_SUGGESTIONS`, and the limit holds under concurrent
  submits. Approved and rejected suggestions do not count toward it.
- **Response:** `id, sub_district_id, category_id, name, address, location,
  operating_hours, estimated_price_range, short_description, attributes, note,
  status, rejection_reason, place_id, reviewed_at, created_at, user{id,name}`.
  `reviewed_by` is **never** exposed.
- **Listing.**
  - `GET /suggestions/mine` lists the caller's own suggestions, newest first;
    on a tie the higher id comes first.
  - `GET /suggestions?status=` (admin) lists oldest first; on a tie the lower id
    comes first. `status` accepts `pending`, `approved`, or `rejected`. It is
    case-sensitive, an empty value means all, and anything else returns
    `400 INVALID_STATUS`.
  - `GET /suggestions/{id}` (admin) returns `INVALID_SUGGESTION_ID` for an
    invalid id and `404 SUGGESTION_NOT_FOUND` for an unknown one.
- **Approve (`POST /suggestions/{id}/approve`, admin).** The body is the
  **final place**, in the same shape as `POST /places`, so the admin can
  correct the suggestion.
  1. A place without a location returns `ErrInvalidLocation` before anything
     is read.
  2. The suggestion must exist (`404`) and be pending; otherwise the response
     is `409 ALREADY_REVIEWED` and no place is created.
  3. The place is created through the **place service**, with full place
     validation. Validation errors return `400 VALIDATION_ERROR` and the
     suggestion stays pending.
  4. The suggestion is marked approved with `place_id`, `reviewed_by`, and
     `reviewed_at`.
  5. If marking fails (another admin reviewed it, or it was deleted), the
     created place is deleted (best effort). Under concurrent approvals exactly
     one place exists.
  6. The suggestion keeps the content the user originally submitted.
- **Reject (`POST /suggestions/{id}/reject {reason?}`, admin).** The body must
  be a JSON object. The reason is trimmed, a blank reason is stored as absent,
  and the limit is 1000 code points (`ErrReasonTooLong`, which returns 400).
  The suggestion must be pending, otherwise `409`. The author sees the reason
  through `/mine`.
- **Reviewed suggestions are immutable.** Any second approve or reject returns
  `409` and changes nothing.
- **Cascades.** Deleting a sub-district or a user deletes their suggestions.
  Deleting an approved place keeps the suggestion approved with
  `place_id = null`.

### 6.7 Image storage (`internal/storage`)

- **Allowed types.** Exactly `image/jpeg` (`.jpg`), `image/png` (`.png`), and
  `image/webp` (`.webp`). Content types with parameters are rejected. Any other
  type returns `ErrInvalidImageType` without touching the store.
- **Presign.** The key is `uploads/{prefix}/{lowercase-uuid}.{ext}` and is
  unique on every call. The URL is a presigned PUT on the **public endpoint**,
  valid for `UploadTTL`. The Content-Type is signed, so uploading with a
  different type returns 403, and expired URLs are refused.
- **Promote(prefix, uploadKey).**
  1. The key must be exactly `uploads/{prefix}/{lowercase-uuid}.{allowed-ext}`.
     Other prefixes, nesting, traversal, already-promoted keys, non-UUID names,
     and missing extensions return `ErrInvalidImageKey` without touching the
     store.
  2. If the object is missing, the result is `ErrImageNotUploaded`.
  3. If the object is larger than `MaxImageSize`, the result is
     `ErrImageTooLarge`. A disallowed type, or a type that does not match the
     extension, returns `ErrInvalidImageType`. In these rejected cases the
     upload is deleted (best effort).
  4. Otherwise the object is copied to `{prefix}/{name}`, the upload is deleted
     (failures ignored), and the new key is returned.
- **PublicURL** is `base + "/" + key`; a trailing slash on the base is
  tolerated.
- **S3 adapter** (Garage locally and in tests). `Stat` returns size and content
  type, or `ErrObjectNotFound`. `Copy` fails when the source is missing.
  `Delete` is idempotent.

### 6.8 Storage cleanup queue (`internal/module/storagecleanup`)

The `STORAGE_DELETIONS` table has columns `key`, `attempts`, `next_attempt_at`,
and `last_error`. Images enter the queue when a place's or a sub-district's image
is deleted or replaced, and when a place or sub-district is deleted. `cmd/cleanup`
runs one batch and exits, so it must be scheduled.

- **`Run`** claims up to 100 due rows and deletes each object.
  - When a delete succeeds, the row is removed from the queue.
  - When a delete fails, the row is rescheduled with backoff
    `15m × 2^attempts`, capped at 24h, and the error message is recorded.
  - A failed delete does not fail the run, and processing continues with the
    next row.
  - If the claim fails, nothing is deleted and the error is returned. If
    removing or rescheduling a row fails, the error is returned.
- **`ClaimDue(limit)`** returns due rows, longest overdue first. Claimed rows are
  hidden from later claims but stay in the queue.
- **`Remove`** is idempotent.
- **`Reschedule`** increments `attempts`, sets `last_error`, and moves
  `next_attempt_at` to now + delay. A delay of 0 makes the row due again.

## 7. Error code catalogue

| Code | Status | Raised by |
| ---- | ------ | --------- |
| `INVALID_BODY` | 400 | Every JSON endpoint (undecodable or wrongly-typed body) |
| `VALIDATION_ERROR` | 400 | Business-rule violations: names, attributes, locations, ratings, lengths, unknown referenced sub-district or category on writes, and image type, key, size, or upload errors |
| `INVALID_PLACE_ID` / `INVALID_SUB_DISTRICT_ID` / `INVALID_CATEGORY_ID` / `INVALID_SUGGESTION_ID` | 400 | Bad path id, or a bad list filter for places |
| `INVALID_LIMIT` / `INVALID_BBOX` | 400 | `GET /places` |
| `INVALID_STATUS` | 400 | `GET /suggestions` |
| `UNAUTHORIZED` | 401 | No authenticated user in the request context |
| `INVALID_CREDENTIALS` / `INVALID_REFRESH_TOKEN` | 401 | Auth |
| `PLACE_NOT_FOUND` / `SUB_DISTRICT_NOT_FOUND` / `CATEGORY_NOT_FOUND` / `SUGGESTION_NOT_FOUND` | 404 | Unknown resource in the path. For `GET /places`, also unknown filter ids |
| `EMAIL_TAKEN` / `SUB_DISTRICT_EXISTS` / `SUB_DISTRICT_IN_USE` / `ALREADY_REVIEWED` | 409 | Conflicts (`ALREADY_REVIEWED` covers both reviews and suggestions) |
| `TOO_MANY_PENDING_SUGGESTIONS` | 429 | Suggestion pending limit |
| `INTERNAL_ERROR` | 500 | Anything unexpected, with no details |

Note that a missing sub-district or category returns **404** when used as a
`GET /places` filter but **400 VALIDATION_ERROR** when referenced in a write
body.

## 8. User stories → backend support

| Story | Backend support |
| ----- | --------------- |
| US-01 Browse pins in the viewport | `GET /places?bbox=…&limit=…`; `total` lets the UI show "N results" when the list is truncated. Clustering and empty states are frontend work |
| US-02 Pin details (6 fields) | Place response: name, category, address/location, operating hours, price range, description. Missing values are `null`, so the UI can show placeholders |
| US-03 Kecamatan dropdown | `GET /subdistricts` (ordered by id) |
| US-04 Kecamatan info modal | Flood, safety, and spending fields plus `static_map_image_url`, with `null` for missing data |
| US-05 Category filter | `GET /categories`; `category_id=1,2` matches any of the listed ids |
| US-06 Keyword search | `q` matches name, address, description, and category name, case-insensitively and literally |
| US-07 Filters stay in sync with the map | All filters and `bbox` combine with AND; `total` reflects the combined filter |

## 9. Where the design docs and the tests disagree

When they disagree, the tests win. The docs record the history:

- **One category per place.** The ERD's many-to-many `PLACE_CATEGORY` was
  replaced: a place has one `category_id`, and categories define typed custom
  fields stored as `attributes`.
- **Kecamatan info is folded into `SUB_DISTRICTS`** (Week 8). The ERD's
  `static_map_image_url` text column is now an uploaded image key. A second
  general `image` slot was added.
- **Coordinates.** The ERD's lat/lng columns became one PostGIS
  `GEOMETRY(Point, 4326)`. The API uses GeoJSON.
- **Paths and error codes.** Week 9–10 reports list `/places/`,
  `/places/categories/`, and lower-case codes such as `invalid_sub_district_id`.
  The tests use `/places`, `/categories`, `/subdistricts`, and UPPER_SNAKE
  codes. The cookie path suggests the API is mounted under `/v1`.
- **Scope.** The ERD and Week 9 report exclude users, reviews, admin, and
  suggestions, and describe a read-only API. Those features now exist (see §4).
- **Voting.** "Upvote / downvote" became 1–5 star reviews.

## 10. Gaps and open questions

These are observations, not decisions. Raise them in PLANNING before relying on
them.

- **No tests cover:** router wiring and URL prefixes, **which routes require
  auth or admin** (place and sub-district writes, image endpoints, suggestion
  admin routes), middleware (`WithAuth` details, logging, recovery, CORS),
  `config`, `httpx`, `jwt`, `postgis`, `database`, `cmd/*` (API, migrate,
  seed), seed data, and the set of categories and fields that the migrations
  define (including the category list order).
- **Admin enforcement** is said to live in the router, but no test verifies it.
- **Refresh-token storage.** Tests use `auth.NewMemoryRefreshStore()`. Whether
  production sessions survive a restart is not specified.
- **Unspecified values:** `UploadTTL`, `MaxImageSize`, how and when the
  cleanup worker runs, and how long access tokens last in production.
- **Unspecified behaviors:** review editing or deletion; a "current user" endpoint; listing users.
- **Frontend (`web/`)** is outside the backend test workflow. US-01
  clustering, placeholders, and filter/viewport sync are untested there.

## Maintaining this file

- Update this file **only after a VERIFICATION phase passes**, so it reflects
  the verified contract. Do not update it for planned or unverified behavior.
- Every behavior stated here must trace to a test. If a test changes the
  contract, update the matching section and remove resolved items from §10.
- Derive content from tests and from the user's approved specification, never
  from production code.
- Bump "Last updated" at the top and add an entry to the log below.

### Change log

| Date | Phase / change | Summary |
| ---- | -------------- | ------- |
| 2026-11-01 | Initial draft | Written from the planning docs and all 19 `*_test.go` files |
| 2026-11-01 | Verification update | Null bodies on JSON endpoints return `INVALID_BODY`; length limits and bare-address email are validated; optional place and sub-district text is trimmed and blank is stored as absent; sub-district image deletes and replacements are queued in `STORAGE_DELETIONS`; a missing refresh cookie clears the cookie; §6.3 error name corrected; resolved §10 items removed |
| 2026-11-05 | Verification update | Categories grew from 5 to 16 (migration `add_perantau_categories`) with themed colors, and `GET /categories` is ordered by id; seed data covers every new category; the web category filter became a Base UI dropdown |
