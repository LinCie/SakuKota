import {
  CloudRain,
  Map as MapIcon,
  ShieldAlert,
  Wallet,
  X,
} from "lucide-react";
import {
  useEffect,
  useRef,
  useState,
  type ComponentType,
  type ReactNode,
} from "react";

import { MissingSlot } from "~/shared/components/missing-slot";
import type { SubDistrict } from "../types";

const rupiah = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});

function InfoSection({
  icon: Icon,
  title,
  children,
}: {
  icon: ComponentType<{
    className?: string;
    strokeWidth?: number;
    "aria-hidden"?: boolean;
  }>;
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="border-ink/25 flex flex-col gap-2 border-t border-dashed pt-4 first:border-t-0 first:pt-0">
      <h3 className="text-ink flex items-center gap-2 text-sm font-bold">
        <Icon
          aria-hidden={true}
          className="text-board size-4"
          strokeWidth={2.5}
        />
        {title}
      </h3>
      {children}
    </section>
  );
}

function AreaMap({ subDistrict }: { subDistrict: SubDistrict }) {
  const [failed, setFailed] = useState(false);
  const src = subDistrict.static_map_image_url;

  if (!src || failed) {
    return (
      <div className="border-ink/30 grid aspect-[8/5] w-full place-items-center rounded-[3px] border-2 border-dashed">
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
      className="border-ink bg-paint-deep aspect-[8/5] w-full rounded-[3px] border-2 object-cover"
    />
  );
}

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
      className="border-ink bg-paint text-ink backdrop:bg-ink/60 m-auto max-h-[calc(100dvh-1.5rem)] w-[min(100%-1.5rem,34rem)] overflow-y-auto overscroll-contain rounded-[6px] border-2 p-0 shadow-[0_24px_60px_rgb(0_0_0/0.35)] motion-safe:transition-[opacity,translate] motion-safe:duration-300 motion-safe:ease-out motion-safe:starting:translate-y-4 motion-safe:starting:opacity-0"
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
                {rupiah.format(spending)}
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
