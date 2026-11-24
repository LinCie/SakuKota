import type * as Leaflet from "leaflet";
import { useEffect, useState, type RefObject } from "react";

// Esri light gray canvas: street geometry and labels without competing POI
// icons, so the painted signs are the only markers on the map.
const TILE_BASE_URL =
  "https://services.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Light_Gray_Base/MapServer/tile/{z}/{y}/{x}";
const TILE_LABELS_URL =
  "https://services.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Light_Gray_Reference/MapServer/tile/{z}/{y}/{x}";
const TILE_ATTRIBUTION =
  'Tiles &copy; Esri &mdash; Esri, HERE, Garmin, &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';

export type LeafletMap = { L: typeof Leaflet; map: Leaflet.Map };

/**
 * Creates a Leaflet map in `containerRef` with the base and labels tiles, and
 * keeps it sized to its container. Resolves to null until the map exists, and
 * again after unmount. `options` is read once, so pass a constant.
 */
export function useLeafletMap(
  containerRef: RefObject<HTMLDivElement | null>,
  options: Leaflet.MapOptions,
): LeafletMap | null {
  const [leaflet, setLeaflet] = useState<LeafletMap | null>(null);

  // Leaflet touches `window`, so it is loaded only in the browser.
  useEffect(() => {
    let cancelled = false;
    let map: Leaflet.Map | undefined;
    let observer: ResizeObserver | undefined;

    void import("leaflet").then((L) => {
      const container = containerRef.current;
      if (cancelled || !container) return;

      // No initial view: the caller sets it, so no tiles load at a throwaway
      // zoom level.
      map = L.map(container, options);
      map.attributionControl.setPrefix(
        '<a href="https://leafletjs.com">Leaflet</a>',
      );
      L.tileLayer(TILE_BASE_URL, {
        maxNativeZoom: 16,
        maxZoom: 19,
        attribution: TILE_ATTRIBUTION,
      }).addTo(map);
      L.tileLayer(TILE_LABELS_URL, { maxNativeZoom: 16, maxZoom: 19 }).addTo(
        map,
      );

      // A container whose size animates reports its view once it settles,
      // not on every frame.
      observer = new ResizeObserver(() =>
        map?.invalidateSize({ debounceMoveend: true }),
      );
      observer.observe(container);

      setLeaflet({ L, map });
    });

    return () => {
      cancelled = true;
      observer?.disconnect();
      map?.remove();
      setLeaflet(null);
    };
  }, [containerRef, options]);

  return leaflet;
}
