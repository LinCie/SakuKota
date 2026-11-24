import { describe, expect, it } from "vitest";

import { BoundaryFileError, readBoundary } from "./geojson";

const square = [
  [
    [106.8, -6.2],
    [106.9, -6.2],
    [106.9, -6.1],
    [106.8, -6.2],
  ],
];
const island = [
  [
    [106.5, -5.7],
    [106.6, -5.7],
    [106.6, -5.6],
    [106.5, -5.7],
  ],
];

const polygon = { type: "Polygon", coordinates: square };
const feature = (geometry: unknown) => ({
  type: "Feature",
  properties: { name: "Menteng" },
  geometry,
});

function read(json: unknown) {
  return readBoundary(JSON.stringify(json));
}

describe("readBoundary", () => {
  it("reads a Polygon as a MultiPolygon", () => {
    expect(read(polygon)).toEqual({
      type: "MultiPolygon",
      coordinates: [square],
    });
  });

  it("reads a MultiPolygon", () => {
    expect(
      read({ type: "MultiPolygon", coordinates: [square, island] }),
    ).toEqual({ type: "MultiPolygon", coordinates: [square, island] });
  });

  it("unwraps a Feature", () => {
    expect(read(feature(polygon)).coordinates).toEqual([square]);
  });

  it("merges every polygon of a FeatureCollection", () => {
    const collection = {
      type: "FeatureCollection",
      features: [
        feature(polygon),
        feature({ type: "MultiPolygon", coordinates: [island] }),
      ],
    };
    expect(read(collection).coordinates).toEqual([square, island]);
  });

  it("drops altitudes", () => {
    const withAltitude = square.map((ring) => ring.map((p) => [...p, 12]));
    expect(
      read({ type: "Polygon", coordinates: withAltitude }).coordinates,
    ).toEqual([square]);
  });

  it.each([
    ["text that is not JSON", "{"],
    ["a JSON array", "[]"],
    ["a Point", JSON.stringify({ type: "Point", coordinates: [106.8, -6.2] })],
    ["a feature without a shape", JSON.stringify(feature(null))],
    [
      "a collection with a line",
      JSON.stringify({
        type: "FeatureCollection",
        features: [
          feature(polygon),
          feature({ type: "LineString", coordinates: square[0] }),
        ],
      }),
    ],
    [
      "an empty collection",
      JSON.stringify({ type: "FeatureCollection", features: [] }),
    ],
    [
      "a position that is not a number",
      JSON.stringify({ type: "Polygon", coordinates: [[["106.8", -6.2]]] }),
    ],
    [
      "projected coordinates",
      JSON.stringify({
        type: "Polygon",
        coordinates: [
          [
            [700000, 9310000],
            [701000, 9310000],
            [701000, 9311000],
            [700000, 9310000],
          ],
        ],
      }),
    ],
  ])("rejects %s", (_, text) => {
    expect(() => readBoundary(text)).toThrow(BoundaryFileError);
  });

  it("rejects a boundary over 1 MB", () => {
    const ring = Array.from({ length: 60_000 }, (_, i) => [
      106.123456789 + i / 1e7,
      -6.123456789,
    ]);
    ring.push(ring[0]);
    expect(() => read({ type: "Polygon", coordinates: [ring] })).toThrow(
      BoundaryFileError,
    );
  });
});
