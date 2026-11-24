import { useNavigate } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useEffect, useEffectEvent, useState } from "react";

import type { SubDistrict } from "~/features/subdistricts/types";
import { Button } from "~/shared/components/ui/button";
import { ConfirmDialog } from "~/shared/components/confirm-dialog";
import { PageHeader } from "~/shared/components/page-header";
import { Pagination } from "~/shared/components/pagination";
import { SearchField } from "~/shared/components/search-field";
import { useDisclosure } from "~/shared/hooks";
import { formatNumber } from "~/shared/format";
import { toastManager } from "~/shared/toast-manager";
import { writeErrorMessage } from "~/shared/write-error";

import type { CategoryDetail, Place } from "../types";
import { PlaceFormDrawer } from "../components/place-form-drawer";
import { PlacesTable } from "../components/places-table";
import { useDeletePlace } from "../queries";

type PlacesPageProps = {
  /** The places on the current page. */
  places: Place[];
  /** The places matching the search, across every page. */
  total: number;
  subDistricts: SubDistrict[];
  categories: CategoryDetail[];
  /** The search the places are filtered by. */
  query: string;
  onQueryChange: (query: string) => void;
  /** The current page, counting from 1. */
  page: number;
  pageCount: number;
  onPageChange: (page: number) => void;
};

/** The places list: search, a paged table with row actions, and the add, edit, and delete flows. */
export function PlacesPage({
  places,
  total,
  subDistricts,
  categories,
  query,
  onQueryChange,
  page,
  pageCount,
  onPageChange,
}: PlacesPageProps) {
  const navigate = useNavigate();
  const [draft, setDraft] = useState(query);

  // Debounce typing into the URL.
  const applyDraft = useEffectEvent(() => onQueryChange(draft.trim()));
  useEffect(() => {
    if (draft.trim() === query) return;
    const timeout = setTimeout(applyDraft, 250);
    return () => clearTimeout(timeout);
  }, [draft, query]);
  const drawer = useDisclosure();
  // Null while adding; the place being edited otherwise.
  const [editing, setEditing] = useState<Place | null>(null);

  const deletePlace = useDeletePlace();
  // The place awaiting delete confirmation.
  const [deleting, setDeleting] = useState<Place | null>(null);

  function closeDelete() {
    setDeleting(null);
    deletePlace.reset();
  }

  function confirmDelete() {
    if (!deleting) return;
    const place = deleting;
    deletePlace.mutate(place.id, {
      onSuccess: () => {
        toastManager.add({ title: `Tempat dihapus: ${place.name}` });
        setDeleting(null);
      },
    });
  }

  const deleteError = deletePlace.error
    ? writeErrorMessage(deletePlace.error)
    : null;

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
          <Button size="sm" onClick={openCreate}>
            <Plus aria-hidden="true" data-icon="inline-start" />
            Tambah tempat
          </Button>
        }
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <SearchField
          value={draft}
          onValueChange={setDraft}
          label="Cari tempat"
          placeholder="Cari nama, alamat, atau kategori"
        />
        <p className="text-muted-foreground text-sm tabular-nums">
          {formatNumber(total)} tempat
        </p>
      </div>

      <PlacesTable
        places={places}
        subDistricts={subDistricts}
        onView={(place) =>
          void navigate({
            to: "/admin/places/$placeId",
            params: { placeId: String(place.id) },
          })
        }
        onEdit={openEdit}
        onDelete={setDeleting}
      />

      {pageCount > 1 ? (
        <Pagination
          page={page}
          pageCount={pageCount}
          onPageChange={onPageChange}
        />
      ) : null}

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
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) closeDelete();
        }}
        title="Hapus tempat?"
        description={`${deleting?.name ?? "Tempat ini"} akan dihapus dari peta. Tindakan ini tidak bisa dibatalkan.`}
        confirmLabel="Hapus tempat"
        pending={deletePlace.isPending}
        error={deleteError}
        onConfirm={confirmDelete}
      />
    </>
  );
}
