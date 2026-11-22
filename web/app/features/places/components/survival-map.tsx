import type * as Leaflet from "leaflet";
import {
  useEffect,
  useEffectEvent,
  useRef,
  type ReactNode,
  type RefObject,
} from "react";

import type { SubDistrict } from "~/features/subdistricts/types";
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
// The map zooms out no further than this, enough to fit a whole kecamatan on a phone.
const MIN_ZOOM = 13;
// ZoomPlate replaces Leaflet's zoom control.
const MAP_OPTIONS: Leaflet.MapOptions = {
  zoomControl: false,
  minZoom: MIN_ZOOM,
};

// Kecamatan outlines. The classes override Leaflet's stroke and fill colors;
// literal so Tailwind generates them. A tooltip's focus listeners make Chrome
// focus a path on mousedown (never by Tab), so its focus ring, a box around
// the whole kecamatan, is hidden.
const SHOWN_OUTLINE: Leaflet.PathOptions = {
  className: "stroke-board fill-board/5",
  weight: 3,
  fillOpacity: 1,
};
const OTHER_OUTLINE: Leaflet.PathOptions = {
  className:
    "stroke-ink-soft fill-transparent outline-none hover:fill-board/10 transition-[fill]",
  weight: 2,
  dashArray: "6 6",
  fillOpacity: 1,
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

/** The fewest places the map loads, and how many before it has a size. */
export const PLACES_LIMIT = 20;

/** The most places the map loads: the server returns no more. */
const MAX_PLACES_LIMIT = 100;

/** The side in pixels of the square each sign gets to itself. */
const SIGN_ROOM = 120;

/** How many places a map of this size loads: one per square of SIGN_ROOM. */
function placesLimit({ x, y }: Leaflet.Point) {
  const fits = Math.floor(x / SIGN_ROOM) * Math.floor(y / SIGN_ROOM);
  return Math.min(MAX_PLACES_LIMIT, Math.max(PLACES_LIMIT, fits));
}

/** The part of the map on show, as the places request filters it. */
export type MapView = {
  /** "west,south,east,north" in degrees. */
  bbox: string;
  /** How many places to load. */
  limit: number;
};

/** Asks the map to teleport to a place. A new object each time, so asking again repeats it. */
export type FlyRequest = { place: Place };

type SurvivalMapProps = {
  /** Places to plant as signs. The map frames them when the kecamatan changes. */
  places: Place[];
  /** The slug of the kecamatan on show; a change triggers a fly-to. */
  focusKey: string;
  /** Whether the places follow a view already; if so, framing the kecamatan is reported too. */
  hasView: boolean;
  /** Every kecamatan. Those with a boundary are outlined, and clicking another one picks it. */
  subDistricts: SubDistrict[];
  onSubDistrictSelect: (slug: string) => void;
  selectedId: number | null;
  onSelect: (placeId: number) => void;
  /** Flies to the place and keeps it clear of the sheet; null until asked. */
  flyRequest: FlyRequest | null;
  /** Reports the view after each pan or zoom, so the places can follow it. Framing a kecamatan is not reported unless `hasView`. */
  onViewChange: (view: MapView) => void;
  /** A bottom sheet that may cover the map; the selected sign is kept above it. */
  sheetRef?: RefObject<HTMLElement | null>;
  /** Overlay rendered above the map, e.g. the empty state. */
  children?: ReactNode;
};

export function SurvivalMap({
  places,
  focusKey,
  hasView,
  subDistricts,
  onSubDistrictSelect,
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
  // True from framing a kecamatan without a view until the map settles; that move is not reported.
  const framingRef = useRef(false);
  const leaflet = useLeafletMap(containerRef, MAP_OPTIONS);
  const onSelectEvent = useEffectEvent(onSelect);
  const onViewChangeEvent = useEffectEvent(onViewChange);
  const onSubDistrictSelectEvent = useEffectEvent(onSubDistrictSelect);

  // The list panel opens and closes beside the map, so it follows its container's size.
  useEffect(() => {
    const container = containerRef.current;
    if (!leaflet || !container) return;
    const observer = new ResizeObserver(() => leaflet.map.invalidateSize());
    observer.observe(container);
    return () => observer.disconnect();
  }, [leaflet]);

  // Registered before the framing effect below, so the move its fitBounds
  // causes is seen, and skipped when the kecamatan's places are loaded.
  useEffect(() => {
    if (!leaflet) return;
    const { map } = leaflet;
    const reportView = () => {
      if (framingRef.current) {
        framingRef.current = false;
        return;
      }
      onViewChangeEvent({
        bbox: bboxParam(map.getBounds()),
        limit: placesLimit(map.getSize()),
      });
    };
    // Dragging stops a flight without a moveend; the drag's own is reported.
    const stopFraming = () => {
      framingRef.current = false;
    };
    map.on("moveend", reportView);
    map.on("dragstart", stopFraming);
    return () => {
      map.off("moveend", reportView);
      map.off("dragstart", stopFraming);
    };
  }, [leaflet]);

  // Frame the selected kecamatan by its boundary, else by its places: jump on
  // first load, fly afterwards.
  useEffect(() => {
    if (!leaflet || focusedKeyRef.current === focusKey) return;
    const { L, map } = leaflet;

    const boundary = subDistricts.find((sd) => sd.slug === focusKey)?.boundary;
    const located = locatedPlaces(places);
    if (!boundary && located.length === 0) {
      if (focusedKeyRef.current === null) {
        framingRef.current = !hasView;
        map.setView(JAKARTA_CENTER, 14);
      }
      focusedKeyRef.current = focusKey;
      return;
    }

    map.invalidateSize();
    const bounds = boundary
      ? L.geoJSON(boundary).getBounds()
      : L.latLngBounds(located.map(({ lat, lng }) => L.latLng(lat, lng)));

    // Set after invalidateSize, whose own moveend (if the size changed) is reported.
    framingRef.current = !hasView;
    if (focusedKeyRef.current === null || prefersReducedMotion()) {
      map.fitBounds(bounds, FIT_OPTIONS);
    } else {
      map.flyToBounds(bounds, { ...FIT_OPTIONS, duration: 0.9 });
    }
    focusedKeyRef.current = focusKey;
  }, [leaflet, focusKey, hasView, places, subDistricts]);

  // Outline every kecamatan: the one on show in bold, the others dashed,
  // named on hover, and picked on click.
  useEffect(() => {
    if (!leaflet) return;
    const { L, map } = leaflet;

    const outlines = subDistricts.flatMap((sd) => {
      if (!sd.boundary) return [];
      const shown = sd.slug === focusKey;
      const outline = L.geoJSON(sd.boundary, {
        style: shown ? SHOWN_OUTLINE : OTHER_OUTLINE,
        // The shown one lets clicks through to the map.
        interactive: !shown,
      });
      if (!shown) {
        outline
          .bindTooltip(`Kec. ${sd.name}`, { sticky: true })
          .on("click", () => onSubDistrictSelectEvent(sd.slug));
      }
      return [outline.addTo(map)];
    });

    return () => {
      for (const outline of outlines) outline.remove();
    };
  }, [leaflet, subDistricts, focusKey]);

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
