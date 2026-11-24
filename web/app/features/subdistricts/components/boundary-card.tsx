import { FileUp, MapIcon } from "lucide-react";
import { useRef, useState } from "react";

import { FormAlert } from "~/shared/components/form-alert";
import { SectionCard } from "~/shared/components/section-card";
import { Button } from "~/shared/components/ui/button";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import { boundaryAccept, BoundaryFileError, readBoundary } from "../geojson";
import { useSetSubDistrictBoundary } from "../queries";
import type { Boundary, SubDistrict } from "../types";
import { BoundaryMap } from "./boundary-map";

type BoundaryCardProps = {
  subDistrict: SubDistrict;
  /** The other kecamatan's boundaries, drawn faintly around this one. */
  others: Boundary[];
};

/**
 * A kecamatan's boundary on a map, and a GeoJSON file picker to replace it.
 * A picked file is previewed first and saved only once confirmed.
 */
export function BoundaryCard({ subDistrict, others }: BoundaryCardProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  // The boundary read from the picked file, awaiting confirmation.
  const [draft, setDraft] = useState<Boundary | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const setBoundary = useSetSubDistrictBoundary();

  async function pick(file: File) {
    setBoundary.reset();
    setDraft(null);
    setFileError(null);
    try {
      setDraft(readBoundary(await file.text()));
    } catch (error) {
      setFileError(
        error instanceof BoundaryFileError
          ? error.message
          : "File tidak terbaca. Coba lagi.",
      );
    }
  }

  function save() {
    if (!draft) return;
    setBoundary.mutate(
      { id: subDistrict.id, boundary: draft },
      {
        onSuccess: () => {
          setDraft(null);
          toastManager.add({ title: "Batas wilayah disimpan" });
        },
      },
    );
  }

  const shape = draft ?? subDistrict.boundary;
  const error =
    fileError ??
    (setBoundary.error ? writeErrorMessage(setBoundary.error) : null);

  return (
    <SectionCard title="Batas wilayah">
      {shape ? (
        <BoundaryMap
          shape={shape}
          previous={draft ? subDistrict.boundary : null}
          others={others}
        />
      ) : (
        <div className="bg-muted text-muted-foreground flex aspect-4/3 w-full flex-col items-center justify-center gap-2 rounded-xl p-3 text-center text-sm">
          <MapIcon aria-hidden="true" className="size-6" />
          Batas wilayah belum ada
        </div>
      )}

      {draft ? (
        <p role="status" className="text-foreground text-sm text-pretty">
          {subDistrict.boundary
            ? "Pratinjau batas baru; garis putus-putus merah adalah batas lama. Periksa bentuknya, lalu simpan."
            : "Pratinjau batas baru. Periksa bentuknya, lalu simpan."}
        </p>
      ) : null}
      {error ? <FormAlert>{error}</FormAlert> : null}

      <div className="flex flex-wrap items-center gap-3">
        {draft ? (
          <>
            <Button size="sm" disabled={setBoundary.isPending} onClick={save}>
              {setBoundary.isPending ? "Menyimpan…" : "Simpan batas"}
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={setBoundary.isPending}
              onClick={() => {
                setDraft(null);
                setBoundary.reset();
              }}
            >
              Batal
            </Button>
          </>
        ) : (
          <Button
            variant="outline"
            size="sm"
            onClick={() => inputRef.current?.click()}
          >
            <FileUp aria-hidden="true" data-icon="inline-start" />
            {subDistrict.boundary ? "Ganti GeoJSON" : "Unggah GeoJSON"}
          </Button>
        )}
        <p className="text-muted-foreground text-xs">
          GeoJSON berisi Polygon atau MultiPolygon, koordinat WGS 84
          (EPSG:4326), maks. 1 MB.
        </p>
      </div>
      <input
        ref={inputRef}
        type="file"
        accept={boundaryAccept}
        hidden
        onChange={(event) => {
          const file = event.target.files?.[0];
          // Cleared so picking the same file again still reads it.
          event.target.value = "";
          if (file) void pick(file);
        }}
      />
    </SectionCard>
  );
}
