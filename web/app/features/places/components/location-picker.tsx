import type * as Leaflet from "leaflet";
import { useEffect, useEffectEvent, useRef } from "react";

import type { Category, Place } from "../types";
import { useLeafletMap } from "../hooks";
import { JAKARTA_CENTER, locatedPlaces, MAP_CANVAS_CLASS } from "./leaflet-map";
import { neighbourIcon, pickedIcon } from "./map-icons";
import { ZoomPlate } from "./zoom-plate";

export type Pin = { lat: number; lng: number };

// A scroll over the map should scroll the page, not zoom the map.
const MAP_OPTIONS: Leaflet.MapOptions = {
  zoomControl: false,
  scrollWheelZoom: false,
};

function round({ lat, lng }: Pin): Pin {
  return { lat: Number(lat.toFixed(6)), lng: Number(lng.toFixed(6)) };
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
  const markerRef = useRef<Leaflet.Marker | null>(null);
  const leaflet = useLeafletMap(containerRef, MAP_OPTIONS);
  const onPickEvent = useEffectEvent(onPick);
  // Only the coordinates move the pin; the effect below depends on these.
  const pinLat = pin?.lat;
  const pinLng = pin?.lng;
  // Reads the latest category without re-planting the pin.
  const pinIconFor = useEffectEvent((L: typeof Leaflet) =>
    pickedIcon(L, category),
  );

  // Frame the chosen kecamatan: its places, or the pin, or Jakarta.
  const frame = useEffectEvent((L: typeof Leaflet, map: Leaflet.Map) => {
    map.invalidateSize();
    const located = locatedPlaces(neighbours);
    if (located.length > 0) {
      map.fitBounds(
        L.latLngBounds(located.map(({ lat, lng }) => L.latLng(lat, lng))),
        { padding: [40, 40], maxZoom: 16 },
      );
    } else if (pin) {
      map.setView([pin.lat, pin.lng], 16);
    } else {
      map.setView(JAKARTA_CENTER, 14);
    }
  });

  useEffect(() => {
    if (leaflet) frame(leaflet.L, leaflet.map);
  }, [leaflet, focusKey]);

  // Draw the neighbours.
  useEffect(() => {
    if (!leaflet) return;
    const { L, map } = leaflet;
    const layer = L.layerGroup().addTo(map);

    for (const { place, lat, lng } of locatedPlaces(neighbours)) {
      L.marker([lat, lng], {
        icon: neighbourIcon(L, place.category),
        interactive: false,
        keyboard: false,
        // Below the pin, even when the layer is redrawn above it.
        zIndexOffset: -1000,
      }).addTo(layer);
    }

    return () => {
      layer.remove();
    };
  }, [leaflet, neighbours]);

  // A tap on the map plants the pin.
  useEffect(() => {
    if (!leaflet) return;
    const { map } = leaflet;
    const onClick = (event: Leaflet.LeafletMouseEvent) =>
      onPickEvent(round(event.latlng));

    map.on("click", onClick);
    return () => {
      map.off("click", onClick);
    };
  }, [leaflet]);

  // Plant, move, or pull the pin.
  useEffect(() => {
    if (!leaflet) return;
    const { L, map } = leaflet;

    if (pinLat === undefined || pinLng === undefined) {
      markerRef.current?.remove();
      markerRef.current = null;
      return;
    }

    const latlng: [number, number] = [pinLat, pinLng];
    if (markerRef.current) {
      markerRef.current.setLatLng(latlng);
    } else {
      const marker = L.marker(latlng, {
        icon: pinIconFor(L),
        draggable: true,
        keyboard: true,
        title: "Lokasi tempat",
        alt: "Lokasi tempat",
      })
        .on("dragend", () => onPickEvent(round(marker.getLatLng())))
        .addTo(map);
      markerRef.current = marker;
    }
    map.panInside(latlng, { padding: [48, 48] });
  }, [leaflet, pinLat, pinLng]);

  // Re-letter the sign when the category changes.
  useEffect(() => {
    if (leaflet && markerRef.current) {
      markerRef.current.setIcon(pickedIcon(leaflet.L, category));
    }
  }, [leaflet, category]);

  return (
    // Inside a drawer, a drag here pans the map instead of swiping the drawer.
    <div
      data-base-ui-swipe-ignore
      className="bg-muted relative isolate h-64 overflow-hidden rounded-xl border"
    >
      <div
        ref={containerRef}
        role="application"
        aria-label="Peta untuk menandai lokasi tempat"
        className={MAP_CANVAS_CLASS}
      />
      <ZoomPlate map={leaflet?.map} />
    </div>
  );
}
