import { Plus, Trash2 } from "lucide-react";

import { FieldMessage } from "~/shared/components/field";
import { Button } from "~/shared/components/ui/button";
import { Input } from "~/shared/components/ui/input";
import {
  NativeSelect,
  NativeSelectOption,
} from "~/shared/components/ui/native-select";

import {
  DAYS,
  MAX_PERIODS_PER_DAY,
  type DayValues,
  type HoursValues,
} from "../hours";

type OperatingHoursInputProps = {
  value: HoursValues;
  onChange: (next: HoursValues) => void;
  error?: string;
};

const modes: { value: DayValues["mode"]; label: string }[] = [
  { value: "closed", label: "Tutup" },
  { value: "24h", label: "24 jam" },
  { value: "hours", label: "Jam tertentu" },
];

/**
 * The week's opening hours: each day is closed, open all day, or open for up to
 * three periods. A place left closed all week has unknown hours.
 */
export function OperatingHoursInput({
  value,
  onChange,
  error,
}: OperatingHoursInputProps) {
  const messageId = "hours-message";

  const setDay = (index: number, next: DayValues) =>
    onChange(value.map((day, i) => (i === index ? next : day)));

  const changeMode = (index: number, mode: DayValues["mode"]) => {
    const { periods } = value[index];
    setDay(index, {
      mode,
      // A day opened by hand starts with a period to fill in.
      periods:
        mode === "hours"
          ? periods.length > 0
            ? periods
            : [{ open: "", close: "" }]
          : [],
    });
  };

  const changePeriod = (
    index: number,
    p: number,
    change: Partial<DayValues["periods"][number]>,
  ) =>
    setDay(index, {
      ...value[index],
      periods: value[index].periods.map((period, i) =>
        i === p ? { ...period, ...change } : period,
      ),
    });

  const removePeriod = (index: number, p: number) => {
    const periods = value[index].periods.filter((_, i) => i !== p);
    setDay(index, { mode: periods.length > 0 ? "hours" : "closed", periods });
  };

  // Most places keep the same hours on several days.
  const firstOpen = value.findIndex((day) => day.mode !== "closed");

  return (
    <div
      role="group"
      aria-labelledby="hours-label"
      aria-describedby={error ? messageId : undefined}
      className="flex flex-col gap-1.5"
    >
      <span id="hours-label" className="text-foreground text-sm font-medium">
        Jam buka
        <span className="text-muted-foreground ml-1.5 font-normal">
          opsional
        </span>
      </span>

      <ul
        className={
          error
            ? "border-destructive flex flex-col divide-y rounded-2xl border"
            : "flex flex-col divide-y rounded-2xl border"
        }
      >
        {DAYS.map(({ day, long }, index) => {
          const dayValue = value[index];
          const selectId = `hours-${day}-mode`;
          return (
            <li key={day} className="flex flex-col gap-2 px-3 py-2.5">
              <div className="flex items-center justify-between gap-3">
                <label htmlFor={selectId} className="text-sm font-medium">
                  {long}
                </label>
                <NativeSelect
                  id={selectId}
                  size="sm"
                  className="w-40"
                  value={dayValue.mode}
                  onChange={(event) =>
                    changeMode(index, event.target.value as DayValues["mode"])
                  }
                >
                  {modes.map((mode) => (
                    <NativeSelectOption key={mode.value} value={mode.value}>
                      {mode.label}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </div>

              {dayValue.mode === "hours" ? (
                <div className="flex flex-col gap-2">
                  {dayValue.periods.map((period, p) => (
                    // Periods have no identity of their own besides their place.
                    // eslint-disable-next-line react-x/no-array-index-key
                    <div key={p} className="flex items-center gap-2">
                      <Input
                        type="time"
                        aria-label={`${long}, jam buka ${p + 1}`}
                        value={period.open}
                        onChange={(event) =>
                          changePeriod(index, p, { open: event.target.value })
                        }
                      />
                      <span
                        aria-hidden="true"
                        className="text-muted-foreground"
                      >
                        –
                      </span>
                      <Input
                        type="time"
                        aria-label={`${long}, jam tutup ${p + 1}`}
                        value={period.close}
                        onChange={(event) =>
                          changePeriod(index, p, { close: event.target.value })
                        }
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={`Hapus jam ${p + 1} hari ${long}`}
                        onClick={() => removePeriod(index, p)}
                      >
                        <Trash2 aria-hidden="true" />
                      </Button>
                    </div>
                  ))}
                  {dayValue.periods.length < MAX_PERIODS_PER_DAY ? (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="self-start"
                      onClick={() =>
                        setDay(index, {
                          ...dayValue,
                          periods: [
                            ...dayValue.periods,
                            { open: "", close: "" },
                          ],
                        })
                      }
                    >
                      <Plus aria-hidden="true" />
                      Tambah jam
                    </Button>
                  ) : null}
                </div>
              ) : null}
            </li>
          );
        })}
      </ul>

      {firstOpen >= 0 ? (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="self-start"
          onClick={() =>
            onChange(
              value.map(() => ({
                mode: value[firstOpen].mode,
                periods: value[firstOpen].periods.map((p) => ({ ...p })),
              })),
            )
          }
        >
          Salin {DAYS[firstOpen].long} ke semua hari
        </Button>
      ) : null}

      <FieldMessage
        id={messageId}
        error={error}
        hint="Biarkan semua hari Tutup jika jam buka belum diketahui. Buka lewat tengah malam? Isi dua hari."
      />
    </div>
  );
}
