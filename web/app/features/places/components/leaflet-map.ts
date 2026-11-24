import type { Place } from "../types";

export const JAKARTA_CENTER: [number, number] = [-6.2088, 106.8456];

/** Class names for a map's container: fills its parent and paints the Leaflet chrome. */
export const MAP_CANVAS_CLASS =
  "bg-muted [&_.leaflet-control-attribution]:bg-card/90 [&_.leaflet-control-attribution]:text-muted-foreground [&_.leaflet-control-attribution_a]:text-primary absolute inset-0 font-sans [&_.leaflet-control-attribution]:rounded-tl-lg [&_.leaflet-control-attribution]:px-2 [&_.leaflet-control-attribution]:text-xs [&_.leaflet-tile-pane]:sepia-[.2] [&_.leaflet-tooltip]:bg-card [&_.leaflet-tooltip]:text-foreground [&_.leaflet-tooltip]:shadow-float [&_.leaflet-tooltip]:rounded-full [&_.leaflet-tooltip]:border-0 [&_.leaflet-tooltip]:px-3 [&_.leaflet-tooltip]:py-1 [&_.leaflet-tooltip]:text-xs [&_.leaflet-tooltip]:font-medium [&_.leaflet-tooltip]:before:hidden";

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
