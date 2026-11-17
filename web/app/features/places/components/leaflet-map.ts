import type { Place } from "../types";

export const JAKARTA_CENTER: [number, number] = [-6.2088, 106.8456];

/** Class names for a map's container: fills its parent and paints the Leaflet chrome. */
export const MAP_CANVAS_CLASS =
  "bg-paint-deep [&_.leaflet-control-attribution]:bg-paint/90 [&_.leaflet-control-attribution]:text-ink-soft [&_.leaflet-control-attribution_a]:text-cat absolute inset-0 font-sans [&_.leaflet-control-attribution]:px-1.5 [&_.leaflet-control-attribution]:text-[10px] [&_.leaflet-tile-pane]:sepia-[.3]";

/** The places that have a location, with their latitude and longitude. */
export function locatedPlaces(places: Place[]) {
  const located = [];
  for (const place of places) {
    if (!place.location) continue;
    // GeoJSON orders coordinates as [longitude, latitude].
    const [lng, lat] = place.location.coordinates;
    located.push({ place, lat, lng });
  }
  return located;
}
