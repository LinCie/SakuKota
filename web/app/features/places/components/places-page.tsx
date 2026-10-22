import { useNavigate, useRouter } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useState } from "react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { Button } from "~/shared/components/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { PageHeader } from "~/shared/components/page-header";
import { SearchField } from "~/shared/components/search-field";
import { useDeletion, useDisclosure, useTableQuery } from "~/shared/hooks";
import { formatNumber } from "~/shared/format";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import { deletePlace } from "../api";
import type { CategoryDetail, Place } from "../types";
import { PlaceFormDrawer } from "./place-form-drawer";
import { PlacesTable } from "./places-table";

type PlacesPageProps = {
  places: Place[];
  subDistricts: SubDistrict[];
  categories: CategoryDetail[];
};

function placeSearchText(place: Place) {
  return [place.name, place.address ?? "", place.category.name].join(" ");
}

/** The places list: search, a table with row actions, and the add, edit, and delete flows. */
export function PlacesPage({
  places,
  subDistricts,
  categories,
}: PlacesPageProps) {
  const router = useRouter();
  const navigate = useNavigate();
  const search = useTableQuery(places, placeSearchText);
  const drawer = useDisclosure();
  // Null while adding; the place being edited otherwise.
  const [editing, setEditing] = useState<Place | null>(null);

  const deletion = useDeletion<Place>(
    async (place) => {
      const result = await deletePlace(place.id);
      if (!result.ok) return writeErrorMessage(result.status, result.error);
      await router.invalidate();
      return null;
    },
    (place) => toastManager.add({ title: `Tempat dihapus: ${place.name}` }),
  );

  function openCreate() {
    setEditing(null);
    drawer.setOpen(true);
  }

  function openEdit(place: Place) {
    setEditing(place);
    drawer.setOpen(true);
  }

  return (
    <>
      <PageHeader
        title="Tempat"
        description="Tempat bertahan hidup di setiap kecamatan."
        actions={
          <Button variant="primary" onClick={openCreate}>
            <Plus aria-hidden="true" className="size-4" strokeWidth={2.5} />
            Tambah tempat
          </Button>
        }
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <SearchField
          value={search.query}
          onValueChange={search.setQuery}
          label="Cari tempat"
          placeholder="Cari nama, alamat, atau kategori"
        />
        <p className="text-ink-soft text-sm tabular-nums">
          {formatNumber(search.rows.length)} dari {formatNumber(places.length)}{" "}
          tempat
        </p>
      </div>

      <PlacesTable
        places={search.rows}
        subDistricts={subDistricts}
        onView={(place) =>
          void navigate({
            to: "/admin/places/$placeId",
            params: { placeId: String(place.id) },
          })
        }
        onEdit={openEdit}
        onDelete={deletion.request}
      />

      <PlaceFormDrawer
        key={editing?.id ?? "new"}
        open={drawer.open}
        onOpenChange={drawer.setOpen}
        place={editing ?? undefined}
        categories={categories}
        subDistricts={subDistricts}
        places={places}
      />

      <ConfirmDialog
        open={deletion.target !== null}
        onOpenChange={(open) => {
          if (!open) deletion.close();
        }}
        title="Hapus tempat?"
        description={`${deletion.target?.name ?? "Tempat ini"} akan dihapus dari peta. Tindakan ini tidak bisa dibatalkan.`}
        confirmLabel="Hapus tempat"
        pending={deletion.pending}
        error={deletion.error}
        onConfirm={() => void deletion.confirm()}
      />
    </>
  );
}
