import {
  CloudRain,
  Image as ImageIcon,
  Map as MapIcon,
  ShieldAlert,
  Wallet,
  X,
} from "lucide-react";
import { useEffect, useRef } from "react";

import { ImageFrame } from "~/shared/components/image-frame";
import { MissingSlot } from "~/shared/components/missing-slot";
import { InfoSection } from "~/shared/components/menu-board";
import { formatRupiah } from "~/shared/format";
import type { SubDistrict } from "../types";
import { AreaMap } from "./area-map";

type KecamatanBoardProps = {
  subDistrict: SubDistrict;
  open: boolean;
  onClose: () => void;
};

export function KecamatanBoard({
  subDistrict,
  open,
  onClose,
}: KecamatanBoardProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  const spending = subDistrict.estimated_daily_spending;

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby="kecamatan-title"
      onClose={onClose}
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
      className="border-ink bg-paint text-ink backdrop:bg-ink/60 m-auto max-h-[calc(100dvh-1.5rem)] w-[min(100%-1.5rem,34rem)] overflow-y-auto overscroll-contain rounded-md border-2 p-0 shadow-[0_24px_60px_rgb(0_0_0/0.35)] motion-safe:transition-[opacity,translate] motion-safe:duration-300 motion-safe:ease-out motion-safe:starting:translate-y-4 motion-safe:starting:opacity-0"
    >
      <header className="bg-board text-paint sticky top-0 z-10 p-1.5">
        <div className="border-sign flex items-center justify-between gap-3 rounded-[3px] border-2 py-2 pr-2 pl-4">
          <h2
            id="kecamatan-title"
            className="font-sign text-sign text-xl leading-tight sm:text-2xl"
          >
            Kec. {subDistrict.name}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup info kecamatan"
            className="text-paint hover:bg-board-deep focus-visible:bg-board-deep focus-visible:outline-sign grid size-9 shrink-0 place-items-center rounded-[3px] transition-colors focus-visible:outline-2"
          >
            <X aria-hidden="true" className="size-5" strokeWidth={2.5} />
          </button>
        </div>
      </header>
      <div aria-hidden="true" className="bg-chili h-1.5" />

      <div className="flex flex-col gap-4 p-4 sm:p-5">
        <InfoSection icon={ImageIcon} title="Foto kecamatan">
          <ImageFrame
            src={subDistrict.image_url}
            alt={`Foto Kecamatan ${subDistrict.name}`}
            missing="Foto tidak tersedia"
            aspect="aspect-8/5"
          />
        </InfoSection>

        <InfoSection icon={CloudRain} title="Riwayat banjir">
          {subDistrict.flood_history ? (
            <p className="text-pretty">{subDistrict.flood_history}</p>
          ) : (
            <MissingSlot>Data tidak tersedia</MissingSlot>
          )}
        </InfoSection>

        <InfoSection icon={ShieldAlert} title="Keamanan">
          {subDistrict.safety_crime_information ? (
            <p className="text-pretty">
              {subDistrict.safety_crime_information}
            </p>
          ) : (
            <MissingSlot>Data tidak tersedia</MissingSlot>
          )}
        </InfoSection>

        <InfoSection icon={Wallet} title="Perkiraan biaya harian">
          {spending !== null ? (
            <p className="flex flex-wrap items-baseline gap-x-2">
              <span className="font-sign text-chili text-4xl leading-none tabular-nums">
                {formatRupiah(spending)}
              </span>
              <span className="text-ink-soft text-sm">per hari, perkiraan</span>
            </p>
          ) : (
            <MissingSlot>Data tidak tersedia</MissingSlot>
          )}
        </InfoSection>

        <InfoSection icon={MapIcon} title="Peta area">
          <AreaMap key={subDistrict.id} subDistrict={subDistrict} />
        </InfoSection>
      </div>
    </dialog>
  );
}
