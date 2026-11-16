// Load test for GET /v1/places replaying a web home page visitor session; run `make seed` first.
// Usage: k6 run -e PROFILE=smoke|load|stress|spike|soak -e RATE=2 -e DURATION=60s -e MAX_VUS=n scripts/k6/get-places.js
import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const PROFILE = __ENV.PROFILE || "load";
const RATE = Number(__ENV.RATE || 2);
// Sessions average about 20s and rarely pass 40s. k6 drops iterations while it
// starts VUs mid-test, so profiles start this many seconds' worth up front.
const SESSION_SECONDS = 30;

// Mirror web/app/features/places (survival-map.tsx, queries.ts, home-page.tsx).
const LIST_PAGE_SIZE = 20;
const SEARCH_DEBOUNCE = 0.25; // seconds
// The server caps every page at this many places.
const SERVER_LIMIT = 40;
// Framing a sub-district: Leaflet's fitBounds padding and zoom cap, and the
// view when there is nothing to frame.
const FIT_PADDING = 56;
const FIT_MAX_ZOOM = 16;
const JAKARTA_CENTER = { lon: 106.8456, lat: -6.2088 };

const MIN_ZOOM = 10;
const MAX_ZOOM = 17;
// Greater Jakarta, where the seed puts places; panning stays inside.
const AREA = { west: 106.68, south: -6.37, east: 106.98, north: -6.1 };
// Map sizes in pixels: desktop beside the list, and a phone.
const VIEWPORTS = [
  { width: 880, height: 720 },
  { width: 390, height: 640 },
];
// Words from the seeded names, addresses, and category names, from rare to
// matching almost everything, plus terms that match nothing.
const SEARCH_TERMS = [
  "warung",
  "apotek",
  "laundry",
  "masjid",
  "klinik",
  "puskesmas",
  "tailor",
  "atm",
  "halte",
  "24 jam",
  "kuningan",
  "karet",
  "mampang",
  "jl.",
  "sushi",
  "indomaret",
];

// Traffic shapes. All but smoke start sessions at a rate that follows the
// stages, in new sessions per second.
const PROFILES = {
  // One visitor, a few sessions: checks the script, seed, and API work.
  smoke: { executor: "per-vu-iterations", vus: 1, iterations: 3 },
  // The expected traffic, held.
  load: arrivalRate([
    { target: RATE, duration: "30s" },
    { target: RATE, duration: __ENV.DURATION || "2m" },
    { target: 0, duration: "15s" },
  ]),
  // Steps up to 8x the expected traffic to find where it breaks.
  stress: arrivalRate([
    ...[1, 2, 4, 8].flatMap((times) => [
      { target: RATE * times, duration: "30s" },
      { target: RATE * times, duration: __ENV.DURATION || "1m" },
    ]),
    { target: 0, duration: "15s" },
  ]),
  // A sudden jump to 10x and back, to see the burst and the recovery.
  spike: arrivalRate([
    { target: RATE, duration: "30s" },
    { target: RATE, duration: "1m" },
    { target: RATE * 10, duration: "10s" },
    { target: RATE * 10, duration: "1m" },
    { target: RATE, duration: "10s" },
    { target: RATE, duration: "2m" },
    { target: 0, duration: "15s" },
  ]),
  // The expected traffic held for long, to surface leaks and slow drift.
  soak: arrivalRate([
    { target: RATE, duration: "1m" },
    { target: RATE, duration: __ENV.DURATION || "1h" },
    { target: 0, duration: "1m" },
  ]),
};

if (!PROFILES[PROFILE]) {
  throw new Error(
    `unknown PROFILE "${PROFILE}"; use ${Object.keys(PROFILES).join(", ")}`,
  );
}

function arrivalRate(stages) {
  const peak = Math.max(...stages.map((stage) => stage.target));
  const maxVUs = Number(__ENV.MAX_VUS) || Math.ceil(peak * SESSION_SECONDS * 2);
  return {
    executor: "ramping-arrival-rate",
    startRate: 0,
    timeUnit: "1s",
    preAllocatedVUs: Math.min(Math.ceil(peak * SESSION_SECONDS), maxVUs),
    maxVUs,
    stages,
  };
}

// Response times split by the API's X-Cache header (hit, miss, or bypass), and
// the share of requests served from the cache.
const cacheDuration = new Trend("places_duration", true);
const cacheHitRate = new Rate("cache_hit");

export const options = {
  scenarios: {
    [PROFILE]: { ...PROFILES[PROFILE], gracefulStop: "1m" },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    checks: ["rate>0.99"],
    "http_req_duration{kind:map}": ["p(95)<500"],
    "http_req_duration{kind:list}": ["p(95)<500"],
    "http_req_duration{search:yes}": ["p(95)<500"],
    "http_req_duration{category:yes}": ["p(95)<500"],
    "places_duration{cache:hit}": ["p(95)<500"],
    "places_duration{cache:miss}": ["p(95)<500"],
  },
};

// Fetches the real sub-district and category ids so filters never 404.
export function setup() {
  const ids = (path) => {
    const res = http.get(`${BASE_URL}/v1/${path}`, {
      tags: { name: `GET /v1/${path}` },
    });
    if (res.status !== 200) {
      throw new Error(`GET /v1/${path}: status ${res.status}`);
    }
    return res.json("data").map((item) => item.id);
  };
  const data = {
    subDistrictIds: ids("subdistricts"),
    categoryIds: ids("places/categories"),
  };
  if (data.subDistrictIds.length === 0 || data.categoryIds.length === 0) {
    throw new Error("no sub-districts or categories; run `make seed` first");
  }
  return data;
}

export default function (data) {
  const state = {
    subDistrictId: pick(data.subDistrictIds),
    categoryIds: [],
    q: "",
    // The map's view; null until the sub-district is framed.
    center: null,
    zoom: null,
    viewport: pick(VIEWPORTS),
    loaded: 0,
    total: 0,
  };
  openSubDistrict(state);

  const steps = randInt(3, 10);
  for (let i = 0; i < steps; i++) {
    sleep(rand(1, 4));
    const roll = Math.random();
    if (roll < 0.45) panOrZoom(state);
    else if (roll < 0.65) scrollList(state);
    else if (roll < 0.8) search(state);
    else if (roll < 0.95) toggleCategory(state, data.categoryIds);
    else switchSubDistrict(state, data.subDistrictIds);
  }
}

function panOrZoom(state) {
  if (Math.random() < 0.6) {
    const { halfLon, halfLat } = halfSpans(state);
    state.center.lon = clamp(
      state.center.lon + rand(-1.6, 1.6) * halfLon,
      AREA.west,
      AREA.east,
    );
    state.center.lat = clamp(
      state.center.lat + rand(-1.6, 1.6) * halfLat,
      AREA.south,
      AREA.north,
    );
  } else {
    // Visitors zoom in to find places more often than they zoom out.
    state.zoom = clamp(state.zoom + pick([-1, 1, 1]), MIN_ZOOM, MAX_ZOOM);
  }
  requestMap(state);
}

// Infinite scroll: loads the next pages until the list runs out.
function scrollList(state) {
  const pages = randInt(1, 4);
  for (let i = 0; i < pages && state.loaded < state.total; i++) {
    if (!requestListPage(state)) return;
    sleep(rand(0.5, 1.5));
  }
}

// Types a term letter by letter. A pause longer than the debounce sends the
// prefix typed so far; the full term is always sent.
function search(state) {
  if (state.q && Math.random() < 0.3) {
    state.q = "";
    refresh(state);
    return;
  }
  const term = pick(SEARCH_TERMS);
  for (let length = 2; length < term.length; length++) {
    if (Math.random() < 0.25) {
      state.q = term.slice(0, length);
      refresh(state);
    }
    sleep(SEARCH_DEBOUNCE);
  }
  state.q = term;
  refresh(state);
}

function toggleCategory(state, categoryIds) {
  if (state.categoryIds.length > 0 && Math.random() < 0.2) {
    state.categoryIds = [];
  } else {
    const id = pick(categoryIds);
    state.categoryIds = state.categoryIds.includes(id)
      ? state.categoryIds.filter((other) => other !== id)
      : [...state.categoryIds, id].sort((a, b) => a - b);
  }
  refresh(state);
}

function switchSubDistrict(state, subDistrictIds) {
  const others = subDistrictIds.filter((id) => id !== state.subDistrictId);
  state.subDistrictId = pick(others.length > 0 ? others : subDistrictIds);
  openSubDistrict(state);
}

// Like picking a kecamatan in the app: loads the sub-district without a bbox, frames the
// places it gets, and then requests that view. The filters stay.
function openSubDistrict(state) {
  state.center = null;
  const body = refresh(state);
  frame(state, body ? body.places : []);
  requestMap(state);
}

// A filter change reloads the map and restarts the list. Returns the map page.
function refresh(state) {
  const body = requestMap(state);
  state.loaded = 0;
  state.total = 0;
  requestListPage(state);
  return body;
}

// Sets the view that fits the located places, like Leaflet's fitBounds with
// the app's padding and zoom cap.
function frame(state, places) {
  const located = places
    .filter((place) => place.location)
    .map((place) => place.location.coordinates);
  if (located.length === 0) {
    state.center = { ...JAKARTA_CENTER };
    state.zoom = 14;
    return;
  }
  const lons = located.map(([lon]) => lon);
  const lats = located.map(([, lat]) => lat);
  const [west, east] = [Math.min(...lons), Math.max(...lons)];
  const [south, north] = [Math.min(...lats), Math.max(...lats)];
  state.center = { lon: (west + east) / 2, lat: (south + north) / 2 };

  // The zoom at which a span of degrees fills the padded pixels.
  const fit = (pixels, degrees) =>
    degrees > 0
      ? Math.log2(((pixels - 2 * FIT_PADDING) * 360) / (256 * degrees))
      : Infinity;
  const cos = Math.cos((state.center.lat * Math.PI) / 180);
  const zoom = Math.min(
    fit(state.viewport.width, east - west),
    fit(state.viewport.height, (north - south) / cos),
  );
  state.zoom = clamp(Math.floor(zoom), MIN_ZOOM, FIT_MAX_ZOOM);
}

// Without a view the map sends no bbox. It never sends a limit.
function requestMap(state) {
  if (!state.center) return getPlaces(state, "map", {});
  return getPlaces(state, "map", { bbox: viewBBox(state) });
}

// Returns whether the page had places.
function requestListPage(state) {
  const body = getPlaces(state, "list", {
    limit: LIST_PAGE_SIZE,
    offset: state.loaded || undefined,
  });
  if (!body) return false;
  state.loaded += body.places.length;
  state.total = body.total;
  return body.places.length > 0;
}

// Sends the request with the session's filters and checks the page against
// them. Returns the page, or null when the request failed.
function getPlaces(state, kind, { limit, offset, bbox }) {
  const q = state.q.trim();
  const params = [
    ["sub_district_id", state.subDistrictId],
    ["category_id", state.categoryIds.join(",")],
    ["q", q],
    ["limit", limit],
    ["offset", offset],
    ["bbox", bbox],
  ]
    .filter(([, value]) => value !== undefined && value !== "")
    .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
    .join("&");

  const res = http.get(`${BASE_URL}/v1/places?${params}`, {
    headers: { "Accept-Encoding": "gzip" },
    tags: {
      name: "GET /v1/places",
      kind,
      search: q ? "yes" : "no",
      category: state.categoryIds.length > 0 ? "yes" : "no",
    },
  });
  if (!check(res, { "status is 200": (r) => r.status === 200 })) return null;

  const cache = (res.headers["X-Cache"] || "none").toLowerCase();
  cacheDuration.add(res.timings.duration, { cache, kind });
  cacheHitRate.add(cache === "hit");

  const body = res.json("data");
  const places = body.places;
  const box = bbox && bbox.split(",").map(Number);
  check(body, {
    "total covers the page": (b) => b.total >= (offset || 0) + places.length,
    "page within limit": () => places.length <= (limit || SERVER_LIMIT),
    "in sub-district": () =>
      places.every((p) => p.sub_district_id === state.subDistrictId),
    "in categories": () =>
      state.categoryIds.length === 0 ||
      places.every((p) => state.categoryIds.includes(p.category.id)),
    "inside bbox": () =>
      !box ||
      places.every((p) => {
        if (!p.location) return false;
        const [lon, lat] = p.location.coordinates;
        return lon >= box[0] && lat >= box[1] && lon <= box[2] && lat <= box[3];
      }),
  });
  return body;
}

// Half the map's width and height in degrees, for Web Mercator tiles of 256
// pixels. Near the equator a degree of latitude is about one of longitude.
function halfSpans({ center, zoom, viewport }) {
  const degreesPerPixel = 360 / (256 * 2 ** zoom);
  return {
    halfLon: (viewport.width / 2) * degreesPerPixel,
    halfLat:
      (viewport.height / 2) *
      degreesPerPixel *
      Math.cos((center.lat * Math.PI) / 180),
  };
}

// The view as "west,south,east,north", rounded outward to 4 decimals like the
// app's bboxParam.
function viewBBox(state) {
  const { center } = state;
  const { halfLon, halfLat } = halfSpans(state);
  const edge = (value, max, round) =>
    (round(clamp(value, -max, max) * 1e4) / 1e4).toFixed(4);
  return [
    edge(center.lon - halfLon, 180, Math.floor),
    edge(center.lat - halfLat, 90, Math.floor),
    edge(center.lon + halfLon, 180, Math.ceil),
    edge(center.lat + halfLat, 90, Math.ceil),
  ].join(",");
}

function rand(min, max) {
  return min + Math.random() * (max - min);
}

function randInt(min, max) {
  return Math.floor(rand(min, max + 1));
}

function pick(items) {
  return items[Math.floor(Math.random() * items.length)];
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
