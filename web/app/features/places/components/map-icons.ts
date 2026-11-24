import type * as Leaflet from "leaflet";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { CircleHelp, type LucideIcon } from "lucide-react";

import type { Category } from "../types";
import { categoryStyle } from "./category-sign";

// Marker HTML is rendered by Leaflet, outside React, so it is built here from
// literal class strings that Tailwind can see.

function iconMarkup(icon: LucideIcon, className: string) {
  return renderToStaticMarkup(createElement(icon, { className }));
}

const FOCUS_RING =
  "rounded-full outline-none focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ring";

// A pin settles into place: it fades in and grows from 60%.
const SETTLING =
  "grid size-full place-items-center motion-safe:transition-[opacity,scale] motion-safe:duration-400 motion-safe:ease-[cubic-bezier(0.16,1,0.3,1)] motion-safe:starting:scale-60 motion-safe:starting:opacity-0";

// Each pin gets a square this wide; it grows to fill it when selected.
const PIN_BOX = 40;

/** A place as a round pin in its category's colour, settling in after `delay`. */
export function placeIcon(
  L: typeof Leaflet,
  category: Category,
  delay: string,
) {
  const { pin, icon } = categoryStyle(category);

  return L.divIcon({
    className: `group ${FOCUS_RING}`,
    iconSize: [PIN_BOX, PIN_BOX],
    iconAnchor: [PIN_BOX / 2, PIN_BOX / 2],
    html: `<span class="${SETTLING} ${delay}"><span class="border-card shadow-pin grid size-8 place-items-center rounded-full border-2 transition-[width,height,scale,background-color,color] duration-200 ease-out group-hover:scale-110 group-data-selected:size-10 group-data-selected:scale-100 group-data-selected:border-3 group-data-selected:bg-foreground group-data-selected:text-card ${pin}">${iconMarkup(icon, "size-4")}</span></span>`,
  });
}

/** The pin being placed: warm ink, like a selected pin. */
export function pickedIcon(L: typeof Leaflet, category: Category | undefined) {
  const icon = category ? categoryStyle(category).icon : CircleHelp;

  return L.divIcon({
    className: FOCUS_RING,
    iconSize: [PIN_BOX, PIN_BOX],
    iconAnchor: [PIN_BOX / 2, PIN_BOX / 2],
    html: `<span class="${SETTLING}"><span class="border-card bg-foreground text-card shadow-pin grid size-10 place-items-center rounded-full border-3">${iconMarkup(icon, "size-5")}</span></span>`,
  });
}

/**
 * The visitor's own position: a telang dot ringed in soft white, with a slow
 * pulse while motion is allowed.
 */
export function userLocationIcon(L: typeof Leaflet) {
  return L.divIcon({
    className: "pointer-events-none",
    iconSize: [22, 22],
    iconAnchor: [11, 11],
    html: `<span class="relative grid size-5.5 place-items-center"><span class="bg-telang/20 absolute -inset-1 rounded-full motion-safe:animate-ping motion-safe:[animation-duration:2s]"></span><span class="bg-telang border-card shadow-pin relative size-5.5 rounded-full border-3"></span></span>`,
  });
}

/** A place already on the map: a small, quiet pin for spotting duplicates. */
export function neighbourIcon(L: typeof Leaflet, category: Category) {
  const { pin, icon } = categoryStyle(category);

  return L.divIcon({
    className: "pointer-events-none",
    iconSize: [24, 24],
    iconAnchor: [12, 12],
    html: `<span class="border-card grid size-6 place-items-center rounded-full border-2 opacity-60 ${pin}">${iconMarkup(icon, "size-3")}</span>`,
  });
}
