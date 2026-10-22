import { useState } from "react";

import { MissingSlot } from "~/shared/components/missing-slot";

import type { SubDistrict } from "../types";

/** The area's static map image, or a drawn slot when it is missing or fails to load. */
export function AreaMap({ subDistrict }: { subDistrict: SubDistrict }) {
  const [failed, setFailed] = useState(false);
  const src = subDistrict.static_map_image_url;

  if (!src || failed) {
    return (
      <div className="border-ink/30 grid aspect-8/5 w-full place-items-center rounded-[3px] border-2 border-dashed">
        <MissingSlot>Peta area tidak tersedia</MissingSlot>
      </div>
    );
  }

  return (
    <img
      src={src}
      alt={`Peta area Kecamatan ${subDistrict.name}`}
      loading="lazy"
      onError={() => setFailed(true)}
      className="border-ink bg-paint-deep aspect-8/5 w-full rounded-[3px] border-2 object-cover"
    />
  );
}
