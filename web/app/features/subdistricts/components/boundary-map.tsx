import type * as Leaflet from "leaflet";
import { useEffect, useRef } from "react";

import { useLeafletMap } from "~/features/places/hooks";
import { MAP_CANVAS_CLASS } from "~/features/places/components/leaflet-map";
import { ZoomPlate } from "~/features/places/components/zoom-plate";

import type { Boundary } from "../types";

// A scroll over the map should scroll the page, not zoom the map.
const MAP_OPTIONS: Leaflet.MapOptions = {
  zoomControl: false,
  scrollWheelZoom: false,
};

const SHAPE_STYLE: Leaflet.PathOptions = {
  className: "stroke-primary fill-primary/10",
  weight: 2,
  fillOpacity: 1,
};
const PREVIOUS_STYLE: Leaflet.PathOptions = {
  className: "stroke-sambal fill-transparent",
  weight: 1.5,
  dashArray: "4 4",
  fillOpacity: 1,
};
const OTHER_STYLE: Leaflet.PathOptions = {
  className: "stroke-muted-foreground/50 fill-transparent",
  weight: 1.5,
  dashArray: "6 6",
  fillOpacity: 1,
};

type BoundaryMapProps = {
  /** The boundary on show; the map frames it. */
  shape: Boundary;
  /** The boundary `shape` would replace, drawn dashed; null if none. */
  previous: Boundary | null;
  /** The other kecamatan's boundaries, faint, to check the fit against. */
  others: Boundary[];
};

/** A Leaflet map that shows a kecamatan's boundary among its neighbours. */
export function BoundaryMap({ shape, previous, others }: BoundaryMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const leaflet = useLeafletMap(containerRef, MAP_OPTIONS);

  useEffect(() => {
    if (!leaflet) return;
    const { L, map } = leaflet;

    const layers = [
      ...others.map((boundary) =>
        L.geoJSON(boundary, { style: OTHER_STYLE, interactive: false }),
      ),
      ...(previous
        ? [L.geoJSON(previous, { style: PREVIOUS_STYLE, interactive: false })]
        : []),
    ];
    const outline = L.geoJSON(shape, {
      style: SHAPE_STYLE,
      interactive: false,
    });
    layers.push(outline);
    for (const layer of layers) layer.addTo(map);

    map.invalidateSize();
    map.fitBounds(outline.getBounds(), { padding: [24, 24] });

    return () => {
      for (const layer of layers) layer.remove();
    };
  }, [leaflet, shape, previous, others]);

  return (
    <div className="bg-muted relative isolate aspect-4/3 overflow-hidden rounded-xl border">
      <div
        ref={containerRef}
        role="img"
        aria-label="Peta batas wilayah kecamatan"
        className={MAP_CANVAS_CLASS}
      />
      <ZoomPlate map={leaflet?.map} />
    </div>
  );
}
