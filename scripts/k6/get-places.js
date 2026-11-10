// Load test for GET /v1/places.
//
//   k6 run scripts/k6/get-places.js
//   k6 run -e BASE_URL=http://localhost:8080 -e VUS=50 -e DURATION=60s scripts/k6/get-places.js
//
// Seed first (`make seed`) so the list has data to return.
import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || "1m",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<500"],
  },
};

// Query strings that exercise the filters GetPlaces supports.
const queries = [
  "",
  "?limit=50",
  "?limit=50&offset=50",
  "?category_id=1",
  "?category_id=1,2",
  "?q=a",
  "?bbox=95,-11,141,6", // all of Indonesia
  "?bbox=106.6,-6.4,107.0,-6.0&limit=100", // Jakarta
];

export default function () {
  const query = queries[Math.floor(Math.random() * queries.length)];
  const res = http.get(`${BASE_URL}/v1/places${query}`, {
    tags: { name: "GET /v1/places" },
  });
  check(res, {
    "status is 200": (r) => r.status === 200,
    "has places": (r) => Array.isArray(r.json("data.places")),
  });
}
