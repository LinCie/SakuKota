import type * as Leaflet from "leaflet";
import { Minus, Plus } from "lucide-react";

import { Button } from "~/shared/components/ui/button";

/** The floating zoom pair that replaces Leaflet's default control. */
export function ZoomPlate({ map }: { map: Leaflet.Map | undefined }) {
  return (
    <div className="bg-card shadow-float absolute top-3 left-3 z-1000 flex flex-col rounded-full p-0.5">
      <Button
        variant="ghost"
        size="icon"
        aria-label="Perbesar peta"
        onClick={() => map?.zoomIn()}
      >
        <Plus aria-hidden="true" className="size-5" />
      </Button>
      <span aria-hidden="true" className="bg-border mx-2.5 h-px" />
      <Button
        variant="ghost"
        size="icon"
        aria-label="Perkecil peta"
        onClick={() => map?.zoomOut()}
      >
        <Minus aria-hidden="true" className="size-5" />
      </Button>
    </div>
  );
}
