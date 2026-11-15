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
  "rounded-[4px] outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cat";

// A standing sign rises from its anchor point when planted.
const STANDING =
  "flex h-full origin-bottom flex-col items-center motion-safe:transition-transform motion-safe:duration-500 motion-safe:ease-[cubic-bezier(0.16,1,0.3,1)] motion-safe:starting:scale-y-0";

/** A place as a painted sign on a post, planted west to east with `delay`. */
export function signIcon(L: typeof Leaflet, category: Category, delay: string) {
  const { solid: plate, icon } = categoryStyle(category);

  return L.divIcon({
    className: `group ${FOCUS_RING}`,
    iconSize: [36, 46],
    iconAnchor: [18, 46],
    html: `<span class="${STANDING} ${delay}"><span class="grid size-9 place-items-center rounded-[3px] border-2 shadow-[0_3px_6px_rgb(26_26_23/0.35)] transition-transform duration-200 ease-out group-hover:-translate-y-0.5 group-data-selected:scale-115 group-data-selected:border-sign group-data-selected:bg-ink group-data-selected:text-sign ${plate}">${iconMarkup(icon, "size-5")}</span><span class="h-2.5 w-0.75 bg-ink"></span></span>`,
  });
}

/** The sign being placed: inverted ink with a yellow code, like a selected sign. */
export function pinIcon(L: typeof Leaflet, category: Category | undefined) {
  const icon = category ? categoryStyle(category).icon : CircleHelp;

  return L.divIcon({
    className: FOCUS_RING,
    iconSize: [36, 46],
    iconAnchor: [18, 46],
    html: `<span class="${STANDING}"><span class="grid size-9 place-items-center rounded-[3px] border-2 border-sign bg-ink text-sign shadow-[0_3px_6px_rgb(26_26_23/0.35)]">${iconMarkup(icon, "size-5")}</span><span class="h-2.5 w-0.75 bg-ink"></span></span>`,
  });
}

/**
 * The visitor's own position: a round painted-blue dot, so it never reads as a
 * square place sign, with a slow ping while motion is allowed.
 */
export function userLocationIcon(L: typeof Leaflet) {
  return L.divIcon({
    className: "pointer-events-none",
    iconSize: [20, 20],
    iconAnchor: [10, 10],
    html: `<span class="relative grid size-5 place-items-center"><span class="bg-cat/40 absolute inset-0 rounded-full motion-safe:animate-ping"></span><span class="bg-cat border-paint ring-ink relative size-4.5 rounded-full border-[3px] ring-2 shadow-[0_3px_6px_rgb(26_26_23/0.35)]"></span></span>`,
  });
}

/** A place already on the map: a small, quiet sign for spotting duplicates. */
export function neighbourIcon(L: typeof Leaflet, category: Category) {
  const { solid, icon } = categoryStyle(category);

  return L.divIcon({
    className: "pointer-events-none",
    iconSize: [24, 30],
    iconAnchor: [12, 30],
    html: `<span class="flex flex-col items-center opacity-60"><span class="grid size-6 place-items-center rounded-[3px] border-2 ${solid}">${iconMarkup(icon, "size-3.5")}</span><span class="h-1.5 w-0.5 bg-ink"></span></span>`,
  });
}
