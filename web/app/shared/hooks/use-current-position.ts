import { useState } from "react";

export type Position = { lat: number; lng: number };

/** Reads the device's current position once, handing it to `onFound`. */
export function useCurrentPosition() {
  const [locating, setLocating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function locate(onFound: (position: Position) => void) {
    setError(null);
    if (!("geolocation" in navigator)) {
      setError("Perangkat ini tidak mendukung lokasi.");
      return;
    }
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        setLocating(false);
        onFound({ lat: coords.latitude, lng: coords.longitude });
      },
      () => {
        setLocating(false);
        setError("Lokasi tidak terbaca. Izinkan akses lokasi, lalu coba lagi.");
      },
      { enableHighAccuracy: true, timeout: 10_000 },
    );
  }

  return { locating, error, locate };
}
