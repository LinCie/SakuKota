import { INDONESIA } from "~/features/places/schemas";

import type { Boundary } from "./types";

// A boundary file is read in the browser and sent as JSON, so it must fit the
// backend's request body limit (internal/middleware/body.go).
const MAX_BOUNDARY_BYTES = 1 << 20;

/** The files the boundary picker offers, as an `accept` attribute value. */
export const boundaryAccept =
  ".geojson,.json,application/geo+json,application/json";

/** Thrown when a boundary file cannot be used, with a message for the admin. */
export class BoundaryFileError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "BoundaryFileError";
  }
}

const notGeoJSON = "File ini bukan GeoJSON yang valid.";
const unreadable = "Koordinat di file ini tidak terbaca.";

type Position = [number, number];
type Polygon = Position[][];

/**
 * Reads a GeoJSON file into a boundary. The file may be a Polygon or
 * MultiPolygon, a Feature of one, or a FeatureCollection of them; every
 * polygon is merged into one MultiPolygon, and altitudes are dropped. Throws
 * a `BoundaryFileError` for anything else, for coordinates outside Indonesia
 * (often a file not in WGS 84), and for a boundary too large to send.
 */
export function readBoundary(text: string): Boundary {
  let json: unknown;
  try {
    json = JSON.parse(text);
  } catch {
    throw new BoundaryFileError(notGeoJSON);
  }

  const polygons = geometriesOf(json).flatMap(polygonsOf);
  if (polygons.length === 0) {
    throw new BoundaryFileError("File ini tidak berisi poligon.");
  }

  const boundary: Boundary = { type: "MultiPolygon", coordinates: polygons };
  if (JSON.stringify(boundary).length > MAX_BOUNDARY_BYTES) {
    throw new BoundaryFileError(
      "Batas ini terlalu rinci (lebih dari 1 MB). Sederhanakan dulu, misalnya di mapshaper.org.",
    );
  }
  return boundary;
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/** The geometries of a GeoJSON object: a collection's features', a feature's, or itself. */
function geometriesOf(json: unknown): unknown[] {
  if (!isObject(json)) throw new BoundaryFileError(notGeoJSON);

  switch (json.type) {
    case "FeatureCollection":
      if (!Array.isArray(json.features)) {
        throw new BoundaryFileError(notGeoJSON);
      }
      return json.features.map(geometryOf);
    case "Feature":
      return [geometryOf(json)];
    default:
      return [json];
  }
}

function geometryOf(feature: unknown): unknown {
  if (!isObject(feature) || feature.type !== "Feature") {
    throw new BoundaryFileError(notGeoJSON);
  }
  return feature.geometry;
}

function polygonsOf(geometry: unknown): Polygon[] {
  if (!isObject(geometry)) {
    throw new BoundaryFileError("Ada fitur tanpa bentuk di file ini.");
  }

  switch (geometry.type) {
    case "Polygon":
      return [polygonOf(geometry.coordinates)];
    case "MultiPolygon":
      return arrayOf(geometry.coordinates).map(polygonOf);
    default:
      throw new BoundaryFileError(
        typeof geometry.type === "string"
          ? `File hanya boleh berisi Polygon atau MultiPolygon, bukan ${geometry.type}.`
          : notGeoJSON,
      );
  }
}

function polygonOf(rings: unknown): Polygon {
  return arrayOf(rings).map((ring) => arrayOf(ring).map(positionOf));
}

function arrayOf(value: unknown): unknown[] {
  if (!Array.isArray(value)) throw new BoundaryFileError(unreadable);
  return value;
}

function positionOf(position: unknown): Position {
  const [lng, lat] = arrayOf(position);
  if (
    typeof lng !== "number" ||
    typeof lat !== "number" ||
    !Number.isFinite(lng) ||
    !Number.isFinite(lat)
  ) {
    throw new BoundaryFileError(unreadable);
  }

  if (
    lng < INDONESIA.minLng ||
    lng > INDONESIA.maxLng ||
    lat < INDONESIA.minLat ||
    lat > INDONESIA.maxLat
  ) {
    throw new BoundaryFileError(
      "Koordinat di luar Indonesia. Ekspor ulang file dengan sistem koordinat WGS 84 (EPSG:4326).",
    );
  }

  // The backend takes [longitude, latitude] only, so any altitude is dropped.
  return [lng, lat];
}
