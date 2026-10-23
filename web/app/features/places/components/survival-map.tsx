import type * as Leaflet from "leaflet";
import {
  useEffect,
  useEffectEvent,
  useRef,
  type ReactNode,
  type RefObject,
} from "react";

import type { Place } from "../types";
import { useLeafletMap } from "../hooks";
import { JAKARTA_CENTER, locatedPlaces, MAP_CANVAS_CLASS } from "./leaflet-map";
import { signIcon } from "./map-icons";
import { ZoomPlate } from "./zoom-plate";

const FIT_OPTIONS = { padding: [56, 56] as [number, number], maxZoom: 16 };
// Below this zoom the map asks for at most MAP_PLACE_LIMIT places.
const MAP_UNLIMITED_ZOOM = 15;
const MAP_PLACE_LIMIT = 25;
// ZoomPlate replaces Leaflet's zoom control.
const MAP_OPTIONS: Leaflet.MapOptions = { zoomControl: false };

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

/** Height of the fixed bottom sheet covering the map; 0 when it doesn't overlap. */
function coveredHeight(
  container: HTMLElement,
  sheet: HTMLElement | null | undefined,
) {
  // Only a fixed (mobile) sheet overlaps the map.
  if (!sheet || getComputedStyle(sheet).position !== "fixed") return 0;
  return Math.max(
    0,
    container.getBoundingClientRect().bottom -
      (window.innerHeight - sheet.offsetHeight),
  );
}

/** The place limit for a zoom level; undefined shows every place. */
function placeLimitForZoom(zoom: number) {
  return zoom < MAP_UNLIMITED_ZOOM ? MAP_PLACE_LIMIT : undefined;
}

type SurvivalMapProps = {
  /** Places to plant as signs. The map frames them when the kecamatan changes. */
  places: Place[];
  /** Changes when the kecamatan changes; triggers a fly-to. */
  focusKey: string;
  selectedId: number | null;
  onSelect: (placeId: number) => void;
  /** Reports the place limit for the current zoom, after each zoom change. */
  onLimitChange: (limit: number | undefined) => void;
  /** A bottom sheet that may cover the map; the selected sign is kept above it. */
  sheetRef?: RefObject<HTMLElement | null>;
  /** Overlay rendered above the map, e.g. the empty state. */
  children?: ReactNode;
};

export function SurvivalMap({
  places,
  focusKey,
  selectedId,
  onSelect,
  onLimitChange,
  sheetRef,
  children,
}: SurvivalMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const markersRef = useRef(new Map<number, Leaflet.Marker>());
  const focusedKeyRef = useRef<string | null>(null);
  const leaflet = useLeafletMap(containerRef, MAP_OPTIONS);
  const onSelectEvent = useEffectEvent(onSelect);
  const onLimitChangeEvent = useEffectEvent(onLimitChange);

  // Registered before the framing effect below, so the zoom its fitBounds
  // causes is reported.
  useEffect(() => {
    if (!leaflet) return;
    const { map } = leaflet;
    const reportLimit = () =>
      onLimitChangeEvent(placeLimitForZoom(map.getZoom()));
    map.on("zoomend", reportLimit);
    return () => {
      map.off("zoomend", reportLimit);
    };
  }, [leaflet]);

  // Frame the selected kecamatan: jump on first load, fly afterwards.
  useEffect(() => {
    if (!leaflet || focusedKeyRef.current === focusKey) return;
    const { L, map } = leaflet;

    const located = locatedPlaces(places);
    if (located.length === 0) {
      if (focusedKeyRef.current === null) map.setView(JAKARTA_CENTER, 14);
      focusedKeyRef.current = focusKey;
      return;
    }

    map.invalidateSize();
    const bounds = L.latLngBounds(
      located.map(({ lat, lng }) => L.latLng(lat, lng)),
    );
    const reduceMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    if (focusedKeyRef.current === null || reduceMotion) {
      map.fitBounds(bounds, FIT_OPTIONS);
    } else {
      map.flyToBounds(bounds, { ...FIT_OPTIONS, duration: 0.9 });
    }
    focusedKeyRef.current = focusKey;
  }, [leaflet, focusKey, places]);

  // Plant new signs and pull the ones that no longer match.
  useEffect(() => {
    if (!leaflet) return;
    const { L, map } = leaflet;
    const markers = markersRef.current;
    const visibleIds = new Set(places.map((place) => place.id));

    for (const [id, marker] of markers) {
      if (!visibleIds.has(id)) {
        marker.remove();
        markers.delete(id);
      }
    }

    const westToEast = locatedPlaces(places)
      .filter(({ place }) => !markers.has(place.id))
      .sort((a, b) => a.lng - b.lng);

    westToEast.forEach(({ place, lat, lng }, index) => {
      const delay = PLANT_DELAYS[Math.min(index, PLANT_DELAYS.length - 1)];
      const marker = L.marker([lat, lng], {
        icon: signIcon(L, place.category, delay),
        title: place.name,
        alt: place.name,
        riseOnHover: true,
        keyboard: true,
      })
        .on("click", () => onSelectEvent(place.id))
        .addTo(map);

      markers.set(place.id, marker);
    });
  }, [leaflet, places]);

  // Invert the selected sign and keep it in view.
  useEffect(() => {
    if (!leaflet) return;
    const markers = markersRef.current;

    for (const [id, marker] of markers) {
      const selected = id === selectedId;
      marker.getElement()?.toggleAttribute("data-selected", selected);
      marker.setZIndexOffset(selected ? 1000 : 0);
    }

    const selectedMarker =
      selectedId !== null ? markers.get(selectedId) : undefined;
    const container = containerRef.current;
    if (selectedMarker && container) {
      leaflet.map.panInside(selectedMarker.getLatLng(), {
        paddingTopLeft: [72, 72],
        paddingBottomRight: [
          72,
          72 + coveredHeight(container, sheetRef?.current),
        ],
      });
    }
  }, [leaflet, selectedId, places, sheetRef]);

  return (
    <div className="absolute inset-0 isolate">
      <div ref={containerRef} className={MAP_CANVAS_CLASS} />

      <ZoomPlate map={leaflet?.map} />

      {children}
    </div>
  );
}
