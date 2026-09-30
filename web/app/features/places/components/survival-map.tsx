import type * as Leaflet from "leaflet";
import { Minus, Plus } from "lucide-react";
import {
  useEffect,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";

import type { Place } from "../types";
import { categoryStyle } from "./category-sign";

const JAKARTA_CENTER: [number, number] = [-6.2088, 106.8456];
const FIT_OPTIONS = { padding: [56, 56] as [number, number], maxZoom: 16 };

// Esri light gray canvas: street geometry and labels without competing POI
// icons, so the painted signs are the only markers on the map.
const TILE_BASE_URL =
  "https://services.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Light_Gray_Base/MapServer/tile/{z}/{y}/{x}";
const TILE_LABELS_URL =
  "https://services.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Light_Gray_Reference/MapServer/tile/{z}/{y}/{x}";
const TILE_ATTRIBUTION =
  'Tiles &copy; Esri &mdash; Esri, HERE, Garmin, &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';

// Signs plant west to east in a quick stagger. Literal class names so
// Tailwind generates them.
const PLANT_DELAYS = [
  "delay-0",
  "delay-40",
  "delay-80",
  "delay-120",
  "delay-160",
  "delay-200",
  "delay-240",
  "delay-280",
  "delay-320",
  "delay-360",
];

function toLatLng(place: Place): [number, number] | null {
  if (!place.location) return null;
  const [longitude, latitude] = place.location.coordinates;
  return [latitude, longitude];
}

/** A place as a small painted sign on a post. */
function signIcon(L: typeof Leaflet, place: Place, delay: string) {
  const category = place.categories[0];
  const style = category ? categoryStyle(category) : null;
  const plate = style?.solid ?? "bg-ink text-paint border-ink";
  const code = style?.code ?? "?";

  return L.divIcon({
    className:
      "group rounded-[4px] outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cat",
    iconSize: [36, 46],
    iconAnchor: [18, 46],
    html: `<span class="flex h-full origin-bottom flex-col items-center motion-safe:transition-transform motion-safe:duration-500 motion-safe:ease-[cubic-bezier(0.16,1,0.3,1)] motion-safe:starting:scale-y-0 ${delay}"><span class="grid size-9 place-items-center rounded-[3px] border-2 font-sign text-base leading-none shadow-[0_3px_6px_rgb(26_26_23/0.35)] transition-transform duration-200 ease-out group-hover:-translate-y-0.5 group-data-selected:scale-115 group-data-selected:border-sign group-data-selected:bg-ink group-data-selected:text-sign ${plate}">${code}</span><span class="h-2.5 w-[3px] bg-ink"></span></span>`,
  });
}

type SurvivalMapProps = {
  /** Places to plant as signs (already filtered). */
  places: Place[];
  /** Places the map frames when the kecamatan changes. */
  focusPlaces: Place[];
  /** Changes when the kecamatan changes; triggers a fly-to. */
  focusKey: string;
  selectedId: number | null;
  onSelect: (placeId: number) => void;
  /** A bottom sheet that may cover the map; the selected sign is kept above it. */
  sheetRef?: RefObject<HTMLElement | null>;
  /** Overlay rendered above the map, e.g. the empty state. */
  children?: ReactNode;
};

export function SurvivalMap({
  places,
  focusPlaces,
  focusKey,
  selectedId,
  onSelect,
  sheetRef,
  children,
}: SurvivalMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const leafletRef = useRef<typeof Leaflet | null>(null);
  const mapRef = useRef<Leaflet.Map | null>(null);
  const markersRef = useRef(new Map<number, Leaflet.Marker>());
  const focusedKeyRef = useRef<string | null>(null);
  const onSelectRef = useRef(onSelect);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    onSelectRef.current = onSelect;
  });

  // Leaflet touches `window`, so it is loaded only in the browser.
  useEffect(() => {
    let cancelled = false;
    let map: Leaflet.Map | undefined;
    let observer: ResizeObserver | undefined;
    const markers = markersRef.current;

    void import("leaflet").then((L) => {
      const container = containerRef.current;
      if (cancelled || !container) return;

      // No initial view: the first kecamatan fit sets it, so no tiles load
      // at a throwaway zoom level.
      map = L.map(container, { zoomControl: false });
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

      observer = new ResizeObserver(() => map?.invalidateSize());
      observer.observe(container);

      leafletRef.current = L;
      mapRef.current = map;
      setReady(true);
    });

    return () => {
      cancelled = true;
      observer?.disconnect();
      map?.remove();
      mapRef.current = null;
      markers.clear();
      focusedKeyRef.current = null;
      setReady(false);
    };
  }, []);

  // Frame the selected kecamatan: jump on first load, fly afterwards.
  useEffect(() => {
    const L = leafletRef.current;
    const map = mapRef.current;
    if (!ready || !L || !map || focusedKeyRef.current === focusKey) return;

    const points = focusPlaces.flatMap((place) => {
      const latLng = toLatLng(place);
      return latLng ? [latLng] : [];
    });
    if (points.length === 0) {
      if (focusedKeyRef.current === null) map.setView(JAKARTA_CENTER, 14);
      focusedKeyRef.current = focusKey;
      return;
    }

    map.invalidateSize();
    const bounds = L.latLngBounds(points);
    const reduceMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    if (focusedKeyRef.current === null || reduceMotion) {
      map.fitBounds(bounds, FIT_OPTIONS);
    } else {
      map.flyToBounds(bounds, { ...FIT_OPTIONS, duration: 0.9 });
    }
    focusedKeyRef.current = focusKey;
  }, [ready, focusKey, focusPlaces]);

  // Plant new signs and pull the ones that no longer match.
  useEffect(() => {
    const L = leafletRef.current;
    const map = mapRef.current;
    if (!ready || !L || !map) return;

    const markers = markersRef.current;
    const visibleIds = new Set(places.map((place) => place.id));

    for (const [id, marker] of markers) {
      if (!visibleIds.has(id)) {
        marker.remove();
        markers.delete(id);
      }
    }

    const westToEast = places
      .filter((place) => !markers.has(place.id) && place.location)
      .sort((a, b) => a.location!.coordinates[0] - b.location!.coordinates[0]);

    westToEast.forEach((place, index) => {
      const delay = PLANT_DELAYS[Math.min(index, PLANT_DELAYS.length - 1)];
      const marker = L.marker(toLatLng(place)!, {
        icon: signIcon(L, place, delay),
        title: place.name,
        alt: place.name,
        riseOnHover: true,
        keyboard: true,
      })
        .on("click", () => onSelectRef.current(place.id))
        .addTo(map);

      markers.set(place.id, marker);
    });
  }, [ready, places]);

  // Invert the selected sign and keep it in view.
  useEffect(() => {
    if (!ready) return;

    for (const [id, marker] of markersRef.current) {
      const selected = id === selectedId;
      marker.getElement()?.toggleAttribute("data-selected", selected);
      marker.setZIndexOffset(selected ? 1000 : 0);
    }

    const selectedMarker =
      selectedId !== null ? markersRef.current.get(selectedId) : undefined;
    const container = containerRef.current;
    if (selectedMarker && container) {
      // Only a fixed (mobile) sheet overlaps the map.
      const sheet = sheetRef?.current;
      const covered =
        sheet && getComputedStyle(sheet).position === "fixed"
          ? Math.max(
              0,
              container.getBoundingClientRect().bottom -
                (window.innerHeight - sheet.offsetHeight),
            )
          : 0;

      mapRef.current?.panInside(selectedMarker.getLatLng(), {
        paddingTopLeft: [72, 72],
        paddingBottomRight: [72, 72 + covered],
      });
    }
  }, [ready, selectedId, places]);

  return (
    <div className="absolute inset-0 isolate">
      <div
        ref={containerRef}
        className="bg-paint-deep [&_.leaflet-control-attribution]:bg-paint/90 [&_.leaflet-control-attribution]:text-ink-soft [&_.leaflet-control-attribution_a]:text-cat absolute inset-0 font-sans [&_.leaflet-control-attribution]:px-1.5 [&_.leaflet-control-attribution]:text-[10px] [&_.leaflet-tile-pane]:sepia-[.3]"
      />

      <div className="border-ink bg-paint absolute top-3 left-3 z-1000 flex flex-col overflow-hidden rounded-[3px] border-2 shadow-[0_3px_8px_rgb(26_26_23/0.25)]">
        <button
          type="button"
          aria-label="Perbesar peta"
          onClick={() => mapRef.current?.zoomIn()}
          className="text-ink hover:bg-sign-soft focus-visible:bg-sign-soft focus-visible:outline-cat grid size-10 place-items-center transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2"
        >
          <Plus aria-hidden="true" className="size-5" strokeWidth={2.5} />
        </button>
        <span aria-hidden="true" className="bg-ink h-0.5" />
        <button
          type="button"
          aria-label="Perkecil peta"
          onClick={() => mapRef.current?.zoomOut()}
          className="text-ink hover:bg-sign-soft focus-visible:bg-sign-soft focus-visible:outline-cat grid size-10 place-items-center transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2"
        >
          <Minus aria-hidden="true" className="size-5" strokeWidth={2.5} />
        </button>
      </div>

      {children}
    </div>
  );
}
