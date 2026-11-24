import { Link } from "@tanstack/react-router";
import { ArrowLeft, Check, X } from "lucide-react";

import { formatAttribute } from "~/features/places/components/attribute-value";
import { CategoryBadge } from "~/features/places/components/category-sign";
import type { CategoryDetail, Place } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { Button, buttonVariants } from "~/shared/components/ui/button";
import { DetailList, type DetailItem } from "~/shared/components/detail-list";
import { MissingSlot, orMissing } from "~/shared/components/missing-slot";
import { PageHeader } from "~/shared/components/page-header";
import { SectionCard } from "~/shared/components/section-card";
import { formatLocalDate } from "~/shared/format";
import { useDisclosure } from "~/shared/hooks";

import { ApproveSuggestionDrawer } from "../components/approve-suggestion-drawer";
import { RejectSuggestionDrawer } from "../components/reject-suggestion-drawer";
import { SuggestionStatusBadge } from "../components/suggestion-status-badge";
import type { Suggestion } from "../types";

type SuggestionDetailPageProps = {
  suggestion: Suggestion;
  subDistricts: SubDistrict[];
  categories: CategoryDetail[];
  places: Place[];
};

/** One suggestion: its review, the place as suggested, and approve and reject. */
export function SuggestionDetailPage({
  suggestion,
  subDistricts,
  categories,
  places,
}: SuggestionDetailPageProps) {
  const approveDrawer = useDisclosure();
  const rejectDrawer = useDisclosure();

  const category = categories.find((c) => c.id === suggestion.category_id);
  const fields = category?.fields ?? [];
  const subDistrict = subDistricts.find(
    (sd) => sd.id === suggestion.sub_district_id,
  );

  const review: DetailItem[] = [
    {
      label: "Status",
      value: <SuggestionStatusBadge status={suggestion.status} />,
    },
    { label: "Pengusul", value: suggestion.user.name },
    {
      label: "Dikirim",
      value: formatLocalDate(new Date(suggestion.created_at)),
    },
  ];
  if (suggestion.reviewed_at) {
    review.push({
      label: "Ditinjau",
      value: formatLocalDate(new Date(suggestion.reviewed_at)),
    });
  }
  if (suggestion.status === "approved") {
    review.push({
      label: "Tempat",
      value:
        suggestion.place_id === null ? (
          <MissingSlot>Tempat sudah dihapus</MissingSlot>
        ) : (
          <Link
            to="/admin/places/$placeId"
            params={{ placeId: String(suggestion.place_id) }}
            className="text-primary font-medium underline underline-offset-2"
          >
            Lihat tempat
          </Link>
        ),
    });
  }
  if (suggestion.status === "rejected") {
    review.push({
      label: "Alasan penolakan",
      value: orMissing(suggestion.rejection_reason, "Tanpa alasan"),
      wide: true,
    });
  }
  review.push({
    label: "Catatan pengusul",
    value: orMissing(suggestion.note, "Tanpa catatan"),
    wide: true,
  });

  const coordinates = suggestion.location?.coordinates;
  const info: DetailItem[] = [
    {
      label: "Kecamatan",
      value: orMissing(subDistrict ? `Kec. ${subDistrict.name}` : null),
    },
    {
      label: "Kategori",
      value: category ? <CategoryBadge category={category} /> : orMissing(null),
    },
    {
      label: "Alamat",
      value: orMissing(suggestion.address, "Alamat tidak tersedia"),
      wide: true,
    },
    {
      label: "Jam buka",
      value: orMissing(suggestion.operating_hours, "Jam buka tidak tersedia"),
    },
    {
      label: "Kisaran harga",
      value: orMissing(
        suggestion.estimated_price_range,
        "Harga tidak tersedia",
      ),
    },
    {
      label: "Koordinat",
      value: coordinates ? (
        `${coordinates[1]}, ${coordinates[0]}`
      ) : (
        <MissingSlot>Lokasi tidak tersedia</MissingSlot>
      ),
    },
    {
      label: "Deskripsi singkat",
      value: orMissing(
        suggestion.short_description,
        "Deskripsi tidak tersedia",
      ),
      wide: true,
    },
  ];

  const attributes: DetailItem[] = fields.map((field) => ({
    label: field.label,
    value: orMissing(formatAttribute(field, suggestion.attributes[field.key])),
    wide: field.data_type === "text",
  }));

  return (
    <>
      <Link
        to="/admin/suggestions"
        className={buttonVariants({
          variant: "ghost",
          size: "sm",
          className: "text-muted-foreground -ml-3 self-start",
        })}
      >
        <ArrowLeft aria-hidden="true" data-icon="inline-start" />
        Daftar usulan
      </Link>

      <PageHeader
        title={suggestion.name}
        description={[
          category?.name,
          subDistrict ? `Kec. ${subDistrict.name}` : null,
        ]
          .filter(Boolean)
          .join(" · ")}
        actions={
          suggestion.status === "pending" ? (
            <>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => rejectDrawer.setOpen(true)}
              >
                <X aria-hidden="true" data-icon="inline-start" />
                Tolak
              </Button>
              <Button size="sm" onClick={() => approveDrawer.setOpen(true)}>
                <Check aria-hidden="true" data-icon="inline-start" />
                Setujui
              </Button>
            </>
          ) : undefined
        }
      />

      <div className="grid gap-5 lg:grid-cols-2">
        <SectionCard title="Tinjauan">
          <DetailList items={review} />
        </SectionCard>
        <SectionCard title="Informasi">
          <DetailList items={info} />
        </SectionCard>
        {fields.length > 0 ? (
          <SectionCard title={`Khusus ${category?.name ?? "kategori"}`}>
            <DetailList items={attributes} />
          </SectionCard>
        ) : null}
      </div>

      {/* Mounted whatever the status, so they can close after a review. */}
      <ApproveSuggestionDrawer
        key={`approve-${suggestion.id}`}
        open={approveDrawer.open}
        onOpenChange={approveDrawer.setOpen}
        suggestion={suggestion}
        categories={categories}
        subDistricts={subDistricts}
        places={places}
      />
      <RejectSuggestionDrawer
        key={`reject-${suggestion.id}`}
        open={rejectDrawer.open}
        onOpenChange={rejectDrawer.setOpen}
        suggestion={suggestion}
      />
    </>
  );
}
