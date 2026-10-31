import { ImageFrame } from "~/shared/components/image-frame";

import type { SubDistrict } from "../types";

/** The area's static map image, or a drawn slot when it is missing or fails to load. */
export function AreaMap({ subDistrict }: { subDistrict: SubDistrict }) {
  return (
    <ImageFrame
      src={subDistrict.static_map_image_url}
      alt={`Peta area Kecamatan ${subDistrict.name}`}
      missing="Peta area tidak tersedia"
      aspect="aspect-8/5"
    />
  );
}
