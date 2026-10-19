import type * as Leaflet from "leaflet";
import { Minus, Plus } from "lucide-react";

import { cn } from "~/shared/cn";
import { menuButtonClass } from "~/shared/components/menu-board";

/** The paint-white zoom plate that replaces Leaflet's default control. */
export function ZoomPlate({ map }: { map: Leaflet.Map | undefined }) {
  return (
    <div className="border-ink bg-paint absolute top-3 left-3 z-1000 flex flex-col overflow-hidden rounded-[3px] border-2 shadow-[0_3px_8px_rgb(26_26_23/0.25)]">
      <button
        type="button"
        aria-label="Perbesar peta"
        onClick={() => map?.zoomIn()}
        className={cn(menuButtonClass, "grid size-10 place-items-center")}
      >
        <Plus aria-hidden="true" className="size-5" strokeWidth={2.5} />
      </button>
      <span aria-hidden="true" className="bg-ink h-0.5" />
      <button
        type="button"
        aria-label="Perkecil peta"
        onClick={() => map?.zoomOut()}
        className={cn(menuButtonClass, "grid size-10 place-items-center")}
      >
        <Minus aria-hidden="true" className="size-5" strokeWidth={2.5} />
      </button>
    </div>
  );
}
