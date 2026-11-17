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
import { LocatePlate } from "./locate-plate";
import { signIcon } from "./map-icons";
import { ZoomPlate } from "./zoom-plate";

const FIT_OPTIONS = { padding: [56, 56] as [number, number], maxZoom: 16 };
// A picked place is shown at least this close.
const PLACE_ZOOM = 18;
// Keeps a selected sign this far from the map's edges.
const SIGN_MARGIN = 72;
// The map zooms out no further than this, a few kilometres across.
const MIN_ZOOM = 16;
// ZoomPlate replaces Leaflet's zoom control.
const MAP_OPTIONS: Leaflet.MapOptions = {
  zoomControl: false,
  minZoom: MIN_ZOOM,
};

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

/** Padding that keeps a point clear of the map's edges and of the bottom sheet. */
function signPadding(
  container: HTMLElement,
  sheet: HTMLElement | null | undefined,
) {
  return {
    paddingTopLeft: [SIGN_MARGIN, SIGN_MARGIN] as [number, number],
    paddingBottomRight: [
      SIGN_MARGIN,
      SIGN_MARGIN + coveredHeight(container, sheet),
    ] as [number, number],
  };
}

function prefersReducedMotion() {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/**
 * The bounds as "west,south,east,north", rounded outward to 4 decimals
 * (about 11 m) so small pans share a request, and clamped to valid
 * coordinates since a zoomed-out map can show past them.
 */
function bboxParam(bounds: Leaflet.LatLngBounds) {
  // West and south round down, east and north round up: outward.
  const west = Math.floor(clamp(bounds.getWest(), 180) * 1e4) / 1e4;
  const south = Math.floor(clamp(bounds.getSouth(), 90) * 1e4) / 1e4;
  const east = Math.ceil(clamp(bounds.getEast(), 180) * 1e4) / 1e4;
  const north = Math.ceil(clamp(bounds.getNorth(), 90) * 1e4) / 1e4;

  return [
    west.toFixed(4),
    south.toFixed(4),
    east.toFixed(4),
    north.toFixed(4),
  ].join(",");
}

/** The value, kept between -limit and limit. */
function clamp(value: number, limit: number) {
  return Math.min(limit, Math.max(-limit, value));
}

/**
 * How many places to load at a zoom level: the server's maximum of 100 when
 * zoomed all the way out, where the view holds the most, down to 40 close in.
 */
function placesLimit(zoom: number) {
  return Math.max(40, 100 - (Math.round(zoom) - MIN_ZOOM) * 20);
}

/** The part of the map on show, as the places request filters it. */
export type MapView = {
  /** "west,south,east,north" in degrees. */
  bbox: string;
  /** How many places to load for the view's zoom level. */
  limit: number;
};

/** Asks the map to teleport to a place. A new object each time, so asking again repeats it. */
export type FlyRequest = { place: Place };

type SurvivalMapProps = {
  /** Places to plant as signs. The map frames them when the kecamatan changes. */
  places: Place[];
  /** Changes when the kecamatan changes; triggers a fly-to. */
  focusKey: string;
  selectedId: number | null;
  onSelect: (placeId: number) => void;
  /** Flies to the place and keeps it clear of the sheet; null until asked. */
  flyRequest: FlyRequest | null;
  /** Reports the view after each pan or zoom, so the places can follow it. */
  onViewChange: (view: MapView) => void;
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
  flyRequest,
  onViewChange,
  sheetRef,
  children,
}: SurvivalMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const markersRef = useRef(new Map<number, Leaflet.Marker>());
  const focusedKeyRef = useRef<string | null>(null);
  // True from a fly request until the map settles; panning to the selected sign meanwhile would cancel it.
  const flyingRef = useRef(false);
  const leaflet = useLeafletMap(containerRef, MAP_OPTIONS);
  const onSelectEvent = useEffectEvent(onSelect);
  const onViewChangeEvent = useEffectEvent(onViewChange);

  // The list panel opens and closes beside the map, so it follows its container's size.
  useEffect(() => {
    const container = containerRef.current;
    if (!leaflet || !container) return;
    const observer = new ResizeObserver(() => leaflet.map.invalidateSize());
    observer.observe(container);
    return () => observer.disconnect();
  }, [leaflet]);

  // Registered before the framing effect below, so the move its fitBounds
  // causes is reported.
  useEffect(() => {
    if (!leaflet) return;
    const { map } = leaflet;
    const reportView = () =>
      onViewChangeEvent({
        bbox: bboxParam(map.getBounds()),
        limit: placesLimit(map.getZoom()),
      });
    map.on("moveend", reportView);
    return () => {
      map.off("moveend", reportView);
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

    if (focusedKeyRef.current === null || prefersReducedMotion()) {
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

  // Teleport to a place picked from the list, like framing a kecamatan.
  useEffect(() => {
    const container = containerRef.current;
    if (!leaflet || !flyRequest || !container) return;
    const { place } = flyRequest;
    if (!place.location) return;
    const { L, map } = leaflet;
    const [lng, lat] = place.location.coordinates;

    map.invalidateSize();
    const bounds = L.latLngBounds([[lat, lng]]);
    const options = {
      ...signPadding(container, sheetRef?.current),
      maxZoom: Math.max(map.getZoom(), PLACE_ZOOM),
    };
    flyingRef.current = true;
    map.once("moveend", () => {
      flyingRef.current = false;
    });
    if (prefersReducedMotion()) {
      map.fitBounds(bounds, options);
    } else {
      map.flyToBounds(bounds, { ...options, duration: 0.9 });
    }
  }, [leaflet, flyRequest, sheetRef]);

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
    if (selectedMarker && container && !flyingRef.current) {
      leaflet.map.panInside(
        selectedMarker.getLatLng(),
        signPadding(container, sheetRef?.current),
      );
    }
  }, [leaflet, selectedId, places, sheetRef]);

  return (
    <div className="absolute inset-0 isolate">
      <div ref={containerRef} className={MAP_CANVAS_CLASS} />

      <ZoomPlate map={leaflet?.map} />
      <LocatePlate leaflet={leaflet} />

      {children}
    </div>
  );
}
