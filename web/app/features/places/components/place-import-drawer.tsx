import { Download, FileUp } from "lucide-react";
import { useRef, useState } from "react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { downloadCsv } from "~/shared/csv";
import { FieldLabel, SelectControl } from "~/shared/components/field";
import { FormAlert } from "~/shared/components/form-alert";
import { FormDrawer } from "~/shared/components/form-drawer";
import { Button } from "~/shared/components/ui/button";
import { formatNumber } from "~/shared/format";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import {
  MAX_IMPORT_PLACES,
  placeCsvColumns,
  placeCsvTemplate,
  readPlaceCsv,
  type PlaceCsvResult,
} from "../csv";
import { useCreatePlaces } from "../queries";
import type { CategoryDetail } from "../types";

type PlaceImportDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
};

// Problems listed before the rest are summed up.
const SHOWN_ERRORS = 10;

/**
 * Adds many places of one category from a CSV file made from that category's
 * template. The file is checked first; the places are saved all at once, or
 * not at all.
 */
export function PlaceImportDrawer({
  open,
  onOpenChange,
  categories,
  subDistricts,
}: PlaceImportDrawerProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [categoryId, setCategoryId] = useState("");
  // The picked file and what was read from it.
  const [file, setFile] = useState<{
    name: string;
    result: PlaceCsvResult;
  } | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const createPlaces = useCreatePlaces();

  const category = categories.find((c) => String(c.id) === categoryId);

  function reset() {
    setCategoryId("");
    setFile(null);
    setFileError(null);
    createPlaces.reset();
  }

  async function pick(picked: File) {
    if (!category) return;
    createPlaces.reset();
    setFileError(null);
    try {
      const text = await picked.text();
      setFile({
        name: picked.name,
        result: readPlaceCsv(text, category, subDistricts),
      });
    } catch {
      setFile(null);
      setFileError("File tidak terbaca. Coba lagi.");
    }
  }

  function submit() {
    if (!file?.result.ok) {
      setFileError(file ? null : "Pilih file CSV dulu.");
      return;
    }
    const count = file.result.places.length;
    createPlaces.mutate(file.result.places, {
      onSuccess: () => {
        toastManager.add({
          title: `${formatNumber(count)} tempat ditambahkan`,
        });
        onOpenChange(false);
      },
    });
  }

  const errors = file && !file.result.ok ? file.result.errors : [];

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      onClosed={reset}
      title="Impor tempat dari CSV"
      description="Tambah banyak tempat satu kategori sekaligus dari template CSV."
      dirty={file !== null}
      formId="place-import-form"
      pending={createPlaces.isPending}
      submitLabel="Impor tempat"
    >
      <form
        id="place-import-form"
        noValidate
        className="flex flex-col gap-5"
        onSubmit={(event) => {
          event.preventDefault();
          submit();
        }}
      >
        <div className="flex flex-col gap-1.5">
          <FieldLabel htmlFor="import-category">Kategori</FieldLabel>
          <SelectControl
            id="import-category"
            value={categoryId}
            onValueChange={(value) => {
              setCategoryId(value);
              // A file is read against its category's columns.
              setFile(null);
              setFileError(null);
              createPlaces.reset();
            }}
            placeholder="Pilih kategori"
            options={categories.map((c) => ({
              value: String(c.id),
              label: c.name,
            }))}
            disabled={createPlaces.isPending}
          />
        </div>

        {category ? (
          <>
            <div className="flex flex-col gap-2">
              <p className="text-foreground text-sm font-medium">Kolom</p>
              <ul className="text-muted-foreground flex flex-col gap-1 text-sm">
                {placeCsvColumns(category).map((column) => (
                  <li key={column.name}>
                    <code className="text-foreground font-mono text-xs">
                      {column.name}
                    </code>
                    {column.required ? null : (
                      <span className="ml-1.5">(opsional)</span>
                    )}{" "}
                    {column.hint}
                  </li>
                ))}
              </ul>
              <p className="text-muted-foreground text-sm text-pretty">
                Satu baris satu tempat, maks. {MAX_IMPORT_PLACES}. Pemisah koma
                atau titik koma.
              </p>
            </div>

            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() =>
                  downloadCsv(
                    `template-tempat-${category.slug}.csv`,
                    placeCsvTemplate(category),
                  )
                }
              >
                <Download aria-hidden="true" data-icon="inline-start" />
                Unduh template
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={createPlaces.isPending}
                onClick={() => inputRef.current?.click()}
              >
                <FileUp aria-hidden="true" data-icon="inline-start" />
                {file ? "Ganti file CSV" : "Pilih file CSV"}
              </Button>
            </div>
          </>
        ) : (
          <p className="text-muted-foreground text-sm text-pretty">
            Pilih kategori untuk melihat kolomnya dan mengunduh template.
          </p>
        )}

        {file?.result.ok ? (
          <p role="status" className="text-foreground text-sm text-pretty">
            {file.name}: {formatNumber(file.result.places.length)} tempat siap
            diimpor.
          </p>
        ) : null}

        {errors.length > 0 ? (
          <FormAlert>
            {file?.name} belum bisa diimpor. Perbaiki lalu pilih lagi.
            <span className="mt-1 flex flex-col font-normal">
              {errors.slice(0, SHOWN_ERRORS).map((error) => (
                <span key={error}>{error}</span>
              ))}
              {errors.length > SHOWN_ERRORS ? (
                <span>
                  …dan {formatNumber(errors.length - SHOWN_ERRORS)} masalah
                  lain.
                </span>
              ) : null}
            </span>
          </FormAlert>
        ) : null}

        {fileError ? <FormAlert>{fileError}</FormAlert> : null}
        {createPlaces.error ? (
          <FormAlert>{writeErrorMessage(createPlaces.error)}</FormAlert>
        ) : null}

        <input
          ref={inputRef}
          type="file"
          accept=".csv,text/csv"
          hidden
          onChange={(event) => {
            const picked = event.target.files?.[0];
            // Cleared so picking the same file again still reads it.
            event.target.value = "";
            if (picked) void pick(picked);
          }}
        />
      </form>
    </FormDrawer>
  );
}
