import type * as Leaflet from "leaflet";
import { LoaderCircle, Locate, LocateFixed } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { cn } from "~/shared/cn";
import { menuButtonClass } from "~/shared/components/menu-board";

import type { LeafletMap } from "../hooks";
import { userLocationIcon } from "./map-icons";

// The visitor is shown at least this close, like a picked place.
const USER_ZOOM = 16;
const NOTE_DURATION = 6000;
const POSITION_OPTIONS: PositionOptions = {
  enableHighAccuracy: true,
  timeout: 10_000,
  maximumAge: 30_000,
};

type LocateStatus = "idle" | "locating" | "located";

function errorMessage(error: GeolocationPositionError) {
  switch (error.code) {
    case error.PERMISSION_DENIED:
      return "Akses lokasi ditolak. Izinkan lokasi untuk situs ini di pengaturan browser.";
    case error.TIMEOUT:
      return "Lokasi terlalu lama ditemukan. Coba lagi.";
    default:
      return "Lokasi belum bisa ditemukan. Coba lagi di tempat terbuka.";
  }
}

/**
 * The paint-white "Lokasi saya" plate: finds the visitor, flies the map there,
 * and marks the spot with a dot and its accuracy circle.
 */
export function LocatePlate({ leaflet }: { leaflet: LeafletMap | null }) {
  const [status, setStatus] = useState<LocateStatus>("idle");
  const [note, setNote] = useState<string | null>(null);
  const markerRef = useRef<Leaflet.Marker | null>(null);
  const accuracyRef = useRef<Leaflet.Circle | null>(null);

  // The marks belong to this map; drop them when it goes.
  useEffect(() => {
    return () => {
      markerRef.current?.remove();
      accuracyRef.current?.remove();
      markerRef.current = null;
      accuracyRef.current = null;
    };
  }, [leaflet]);

  // A note clears itself after a while.
  useEffect(() => {
    if (!note) return;
    const timeout = setTimeout(() => setNote(null), NOTE_DURATION);
    return () => clearTimeout(timeout);
  }, [note]);

  function showPosition({ coords }: GeolocationPosition) {
    if (!leaflet) return;
    const { L, map } = leaflet;
    const latLng = L.latLng(coords.latitude, coords.longitude);

    if (markerRef.current && accuracyRef.current) {
      markerRef.current.setLatLng(latLng);
      accuracyRef.current.setLatLng(latLng).setRadius(coords.accuracy);
    } else {
      accuracyRef.current = L.circle(latLng, {
        radius: coords.accuracy,
        className: "fill-cat stroke-cat",
        weight: 1.5,
        opacity: 0.6,
        fillOpacity: 0.12,
        interactive: false,
      }).addTo(map);
      markerRef.current = L.marker(latLng, {
        icon: userLocationIcon(L),
        interactive: false,
        keyboard: false,
        zIndexOffset: -1000,
      }).addTo(map);
    }

    const zoom = Math.max(map.getZoom(), USER_ZOOM);
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      map.setView(latLng, zoom);
    } else {
      map.flyTo(latLng, zoom, { duration: 0.9 });
    }
    setStatus("located");
  }

  function locate() {
    // Ignored rather than disabled while locating, so the button keeps focus.
    if (status === "locating") return;
    setNote(null);
    if (!("geolocation" in navigator)) {
      setNote("Browser ini tidak bisa membagikan lokasi.");
      return;
    }
    setStatus("locating");
    navigator.geolocation.getCurrentPosition(
      showPosition,
      (error) => {
        setStatus(markerRef.current ? "located" : "idle");
        setNote(errorMessage(error));
      },
      POSITION_OPTIONS,
    );
  }

  const locating = status === "locating";
  let Icon = Locate;
  if (locating) Icon = LoaderCircle;
  else if (status === "located") Icon = LocateFixed;

  return (
    <div className="pointer-events-none absolute right-3 bottom-6 z-1000 flex flex-col items-end gap-2">
      <p
        role="status"
        className="bg-ink text-paint pointer-events-auto max-w-[min(100vw-1.5rem,18rem)] rounded-[5px] px-3 py-2.5 text-sm font-semibold text-pretty shadow-[0_3px_8px_rgb(26_26_23/0.25)] empty:hidden"
      >
        {note}
      </p>
      <button
        type="button"
        onClick={locate}
        disabled={!leaflet}
        aria-label="Tampilkan lokasi saya"
        aria-busy={locating}
        title="Lokasi saya"
        className={cn(
          menuButtonClass,
          "border-ink bg-paint pointer-events-auto grid size-11 place-items-center rounded-[3px] border-2 shadow-[0_3px_8px_rgb(26_26_23/0.25)] disabled:opacity-50 aria-busy:cursor-progress md:size-10",
        )}
      >
        <Icon
          aria-hidden="true"
          className={cn(
            "size-5",
            locating && "motion-safe:animate-spin",
            status === "located" && "text-cat",
          )}
          strokeWidth={2.5}
        />
      </button>
    </div>
  );
}
