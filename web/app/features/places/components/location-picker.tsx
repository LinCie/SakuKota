import type * as Leaflet from "leaflet";
import { useEffect, useRef, useState } from "react";

import type { Category, Place } from "../types";
import { categoryStyle } from "./category-sign";
import {
  JAKARTA_CENTER,
  TILE_ATTRIBUTION,
  TILE_BASE_URL,
  TILE_LABELS_URL,
} from "./survival-map";
import { ZoomPlate } from "./zoom-plate";

export type Pin = { lat: number; lng: number };

function round({ lat, lng }: Pin): Pin {
  return { lat: Number(lat.toFixed(6)), lng: Number(lng.toFixed(6)) };
}

/** The sign being placed: inverted ink with a yellow code, like a selected pin. */
function pinIcon(L: typeof Leaflet, category: Category | undefined) {
  const code = category ? categoryStyle(category).code : "?";

  return L.divIcon({
    className:
      "rounded-[4px] outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cat",
    iconSize: [36, 46],
    iconAnchor: [18, 46],
    html: `<span class="flex h-full origin-bottom flex-col items-center motion-safe:transition-transform motion-safe:duration-500 motion-safe:ease-[cubic-bezier(0.16,1,0.3,1)] motion-safe:starting:scale-y-0"><span class="grid size-9 place-items-center rounded-[3px] border-2 border-sign bg-ink font-sign text-base leading-none text-sign shadow-[0_3px_6px_rgb(26_26_23/0.35)]">${code}</span><span class="h-2.5 w-0.75 bg-ink"></span></span>`,
  });
}

/** A place already on the map: a small, quiet sign for spotting duplicates. */
function neighbourIcon(L: typeof Leaflet, place: Place) {
  const { solid, code } = categoryStyle(place.category);

  return L.divIcon({
    className: "pointer-events-none",
    iconSize: [24, 30],
    iconAnchor: [12, 30],
    html: `<span class="flex flex-col items-center opacity-60"><span class="grid size-6 place-items-center rounded-[3px] border-2 font-sign text-xs leading-none ${solid}">${code}</span><span class="h-1.5 w-0.5 bg-ink"></span></span>`,
  });
}

type LocationPickerProps = {
  /** The chosen location, or null while none is set. */
  pin: Pin | null;
  onPick: (pin: Pin) => void;
  /** Places already in the chosen kecamatan; memoize, it triggers a redraw. */
  neighbours: Place[];
  /** The place's category, which sets the code on the sign. */
  category: Category | undefined;
  /** Changes with the chosen kecamatan; reframes the map. */
  focusKey: string;
};

/**
 * A Leaflet map where a tap plants the place's sign; the sign can be dragged to
 * fine-tune it. The latitude and longitude fields stay in sync through `pin`.
 */
export function LocationPicker({
  pin,
  onPick,
  neighbours,
  category,
  focusKey,
}: LocationPickerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const leafletRef = useRef<typeof Leaflet | null>(null);
  const mapRef = useRef<Leaflet.Map | null>(null);
  const markerRef = useRef<Leaflet.Marker | null>(null);
  const markerCategoryRef = useRef<number | undefined>(undefined);
  const neighbourLayerRef = useRef<Leaflet.LayerGroup | null>(null);
  const onPickRef = useRef(onPick);
  const neighboursRef = useRef(neighbours);
  const pinRef = useRef(pin);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    onPickRef.current = onPick;
    neighboursRef.current = neighbours;
    pinRef.current = pin;
  });

  // Leaflet touches `window`, so it is loaded only in the browser.
  useEffect(() => {
    let cancelled = false;
    let map: Leaflet.Map | undefined;
    let observer: ResizeObserver | undefined;

    void import("leaflet").then((L) => {
      const container = containerRef.current;
      if (cancelled || !container) return;

      map = L.map(container, { zoomControl: false, scrollWheelZoom: false });
      map.setView(JAKARTA_CENTER, 14);
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
      map.on("click", (event) => onPickRef.current(round(event.latlng)));

      observer = new ResizeObserver(() => map?.invalidateSize());
      observer.observe(container);

      neighbourLayerRef.current = L.layerGroup().addTo(map);
      leafletRef.current = L;
      mapRef.current = map;
      setReady(true);
    });

    return () => {
      cancelled = true;
      observer?.disconnect();
      map?.remove();
      mapRef.current = null;
      markerRef.current = null;
      markerCategoryRef.current = undefined;
      neighbourLayerRef.current = null;
      setReady(false);
    };
  }, []);

  // Frame the chosen kecamatan: its places, or the pin, or Jakarta.
  useEffect(() => {
    const L = leafletRef.current;
    const map = mapRef.current;
    if (!ready || !L || !map) return;

    const points = neighboursRef.current.flatMap((place) =>
      place.location
        ? [
            [place.location.coordinates[1], place.location.coordinates[0]] as [
              number,
              number,
            ],
          ]
        : [],
    );
    const current = pinRef.current;

    map.invalidateSize();
    if (points.length > 0) {
      map.fitBounds(L.latLngBounds(points), { padding: [40, 40], maxZoom: 16 });
    } else if (current) {
      map.setView([current.lat, current.lng], 16);
    } else {
      map.setView(JAKARTA_CENTER, 14);
    }
  }, [ready, focusKey]);

  // Draw the neighbours.
  useEffect(() => {
    const L = leafletRef.current;
    const layer = neighbourLayerRef.current;
    if (!ready || !L || !layer) return;

    layer.clearLayers();
    for (const place of neighbours) {
      if (!place.location) continue;
      const [lng, lat] = place.location.coordinates;
      L.marker([lat, lng], {
        icon: neighbourIcon(L, place),
        interactive: false,
        keyboard: false,
      }).addTo(layer);
    }
  }, [ready, neighbours]);

  // Plant, move, or pull the pin.
  useEffect(() => {
    const L = leafletRef.current;
    const map = mapRef.current;
    if (!ready || !L || !map) return;

    if (!pin) {
      markerRef.current?.remove();
      markerRef.current = null;
      return;
    }

    const latlng: [number, number] = [pin.lat, pin.lng];
    if (markerRef.current) {
      markerRef.current.setLatLng(latlng);
    } else {
      const marker = L.marker(latlng, {
        icon: pinIcon(L, category),
        draggable: true,
        keyboard: true,
        title: "Lokasi tempat",
        alt: "Lokasi tempat",
      })
        .on("dragend", () => onPickRef.current(round(marker.getLatLng())))
        .addTo(map);
      markerRef.current = marker;
      markerCategoryRef.current = category?.id;
    }
    map.panInside(latlng, { padding: [48, 48] });
    // The category only restyles the sign; see the effect below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready, pin?.lat, pin?.lng]);

  // Re-letter the sign when the category changes.
  useEffect(() => {
    const L = leafletRef.current;
    const marker = markerRef.current;
    if (!ready || !L || !marker || markerCategoryRef.current === category?.id) {
      return;
    }
    marker.setIcon(pinIcon(L, category));
    markerCategoryRef.current = category?.id;
  }, [ready, category]);

  return (
    <div className="border-ink bg-paint-deep relative isolate h-64 overflow-hidden rounded-[3px] border-2">
      <div
        ref={containerRef}
        role="application"
        aria-label="Peta untuk menandai lokasi tempat"
        className="[&_.leaflet-control-attribution]:bg-paint/90 [&_.leaflet-control-attribution]:text-ink-soft [&_.leaflet-control-attribution_a]:text-cat absolute inset-0 font-sans [&_.leaflet-control-attribution]:px-1.5 [&_.leaflet-control-attribution]:text-[10px] [&_.leaflet-tile-pane]:sepia-[.3]"
      />
      <ZoomPlate
        onZoomIn={() => mapRef.current?.zoomIn()}
        onZoomOut={() => mapRef.current?.zoomOut()}
      />
    </div>
  );
}
