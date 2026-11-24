import {
  CloudRain,
  Image as ImageIcon,
  ShieldAlert,
  Wallet,
  X,
} from "lucide-react";

import { MediaGallery } from "~/shared/components/media-gallery";
import { InfoSection } from "~/shared/components/menu-board";
import { MissingSlot } from "~/shared/components/missing-slot";
import { Button } from "~/shared/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "~/shared/components/ui/dialog";
import { formatRupiah } from "~/shared/format";
import type { SubDistrict } from "../types";

type KecamatanBoardProps = {
  subDistrict: SubDistrict;
  open: boolean;
  onClose: () => void;
};

/** The kecamatan's info in a dialog: photos, floods, safety, and daily cost. */
export function KecamatanBoard({
  subDistrict,
  open,
  onClose,
}: KecamatanBoardProps) {
  const spending = subDistrict.estimated_daily_spending;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
    >
      <DialogContent
        showCloseButton={false}
        className="block max-h-[calc(100dvh-1.5rem)] max-w-[calc(100%-1.5rem)] overflow-y-auto overscroll-contain p-0 text-base sm:max-w-136"
      >
        <header className="bg-popover sticky top-0 z-10 flex items-center justify-between gap-3 border-b py-3 pr-3 pl-5 sm:pl-6">
          <DialogTitle className="text-xl leading-tight">
            Kec. {subDistrict.name}
          </DialogTitle>
          <DialogClose
            render={
              <Button
                variant="ghost"
                size="icon"
                aria-label="Tutup info kecamatan"
              />
            }
          >
            <X aria-hidden="true" className="size-5" />
          </DialogClose>
        </header>

        <div className="flex flex-col p-5 sm:p-6">
          <InfoSection icon={ImageIcon} title="Foto kecamatan">
            <MediaGallery
              medias={subDistrict.medias}
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
                <span className="text-foreground text-2xl leading-none font-semibold tabular-nums">
                  {formatRupiah(spending)}
                </span>
                <span className="text-muted-foreground text-sm">
                  per hari, perkiraan
                </span>
              </p>
            ) : (
              <MissingSlot>Data tidak tersedia</MissingSlot>
            )}
          </InfoSection>
        </div>
      </DialogContent>
    </Dialog>
  );
}
