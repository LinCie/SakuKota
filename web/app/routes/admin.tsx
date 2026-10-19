import { Tabs } from "@base-ui/react/tabs";
import {
  createFileRoute,
  Link,
  redirect,
  useRouter,
} from "@tanstack/react-router";
import { ArrowLeft, Check } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

import { AccountControl } from "~/features/auth/components/account-control";
import { useSession } from "~/features/auth/hooks";
import { getSession, isAdmin } from "~/features/auth/session";
import {
  PlacesPanel,
  SubDistrictsPanel,
} from "~/features/admin/components/admin-panels";
import { getCategories, getPlaces } from "~/features/places/api";
import { PlaceSheet } from "~/features/places/components/place-sheet";
import type { Place } from "~/features/places/types";
import { getSubDistricts } from "~/features/subdistricts/api";
import { SubDistrictSheet } from "~/features/subdistricts/components/subdistrict-sheet";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { ErrorBoard } from "~/shared/components/error-board";
import { signButtonClass } from "~/shared/components/menu-board";
import { Wordmark } from "~/shared/components/wordmark";

type AdminSearch = {
  /** Absent means the places tab. */
  tab?: "kecamatan";
};

export const Route = createFileRoute("/admin")({
  validateSearch: (search: Record<string, unknown>): AdminSearch => ({
    tab: search.tab === "kecamatan" ? "kecamatan" : undefined,
  }),
  beforeLoad: ({ location }) => {
    if (!getSession()) {
      throw redirect({
        to: "/masuk",
        search: { kembali: location.href },
        replace: true,
      });
    }
  },
  loader: async () => {
    const [categories, subDistricts, places] = await Promise.all([
      getCategories(),
      getSubDistricts(),
      getPlaces(),
    ]);
    if (!categories.ok) throw new Error(categories.error.message);
    if (!subDistricts.ok) throw new Error(subDistricts.error.message);
    if (!places.ok) throw new Error(places.error.message);

    return {
      categories: categories.data,
      subDistricts: subDistricts.data,
      places: places.data,
    };
  },
  head: () => ({ meta: [{ title: "Admin · Rantau" }] }),
  component: Admin,
  errorComponent: AdminError,
});

// -----------------------------------------------------------------------------
// Page
// -----------------------------------------------------------------------------

function AdminHeader() {
  return (
    <header className="bg-board text-paint">
      <div className="flex items-center justify-between gap-3 px-3 py-3 md:px-5">
        <div className="flex min-w-0 items-center gap-3">
          <Wordmark tagline={false} />
          <span className="bg-ink font-sign text-sign rounded-[3px] px-2 pt-1.5 pb-1 text-sm leading-none">
            Admin
          </span>
        </div>
        <AccountControl className="shrink-0" />
      </div>
      <div aria-hidden="true" className="bg-chili h-1.5" />
    </header>
  );
}

const quietLink =
  "text-ink hover:bg-sign-soft focus-visible:bg-sign-soft focus-visible:outline-cat inline-flex h-10 items-center gap-2 rounded-[3px] px-2 text-sm font-semibold transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2";

function BackToMap() {
  return (
    <Link to="/" className={quietLink}>
      <ArrowLeft aria-hidden="true" className="size-4" strokeWidth={2.5} />
      Kembali ke peta
    </Link>
  );
}

function Admin() {
  const session = useSession();
  const { categories, subDistricts, places } = Route.useLoaderData();
  const { tab } = Route.useSearch();
  const navigate = Route.useNavigate();
  const router = useRouter();

  const [placeOpen, setPlaceOpen] = useState(false);
  const [subDistrictOpen, setSubDistrictOpen] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [highlight, setHighlight] = useState<string | null>(null);

  // The notice and the new row's highlight fade after a moment.
  useEffect(() => {
    if (!notice) return;
    const timeout = setTimeout(() => setNotice(null), 6000);
    return () => clearTimeout(timeout);
  }, [notice]);
  useEffect(() => {
    if (!highlight) return;
    const timeout = setTimeout(() => setHighlight(null), 2500);
    return () => clearTimeout(timeout);
  }, [highlight]);

  if (!isAdmin(session)) {
    return <AdminGate signedIn={session !== null} />;
  }

  async function placeCreated(place: Place) {
    await router.invalidate();
    setNotice(`Tempat ditambahkan: ${place.name}`);
    setHighlight(`tempat-${place.id}`);
  }

  async function subDistrictCreated(subDistrict: SubDistrict) {
    await router.invalidate();
    setNotice(`Kecamatan ditambahkan: ${subDistrict.name}`);
    setHighlight(`kecamatan-${subDistrict.id}`);
  }

  return (
    <div className="bg-paint text-ink flex min-h-dvh flex-col">
      <AdminHeader />

      <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col">
        <h1 className="sr-only">Admin Rantau</h1>

        <div className="px-1 pt-2 md:px-3">
          <BackToMap />
        </div>

        <Tabs.Root
          value={tab ?? "tempat"}
          onValueChange={(value) =>
            void navigate({
              search: { tab: value === "kecamatan" ? "kecamatan" : undefined },
              replace: true,
              resetScroll: false,
            })
          }
          className="flex flex-1 flex-col"
        >
          <Tabs.List
            aria-label="Jenis data"
            className="grid grid-cols-2 gap-2 px-3 pt-1 md:px-5"
          >
            <Tab value="tempat">Tempat</Tab>
            <Tab value="kecamatan">Kecamatan</Tab>
          </Tabs.List>

          <div role="status" className="px-3 pt-3 empty:hidden md:px-5">
            {notice ? (
              <p className="border-ink bg-sign-soft flex items-center gap-2 rounded-[3px] border-2 px-3 py-2.5 text-sm font-semibold">
                <Check
                  aria-hidden="true"
                  className="size-4 shrink-0"
                  strokeWidth={2.5}
                />
                {notice}
              </p>
            ) : null}
          </div>

          <Tabs.Panel
            value="tempat"
            className="flex flex-1 flex-col outline-none"
          >
            <PlacesPanel
              places={places}
              subDistricts={subDistricts}
              highlightId={highlight}
              onAdd={() => setPlaceOpen(true)}
            />
          </Tabs.Panel>
          <Tabs.Panel
            value="kecamatan"
            className="flex flex-1 flex-col outline-none"
          >
            <SubDistrictsPanel
              places={places}
              subDistricts={subDistricts}
              highlightId={highlight}
              onAdd={() => setSubDistrictOpen(true)}
            />
          </Tabs.Panel>
        </Tabs.Root>
      </main>

      <PlaceSheet
        open={placeOpen}
        onOpenChange={setPlaceOpen}
        categories={categories}
        subDistricts={subDistricts}
        places={places}
        onCreated={placeCreated}
      />
      <SubDistrictSheet
        open={subDistrictOpen}
        onOpenChange={setSubDistrictOpen}
        onCreated={subDistrictCreated}
      />
    </div>
  );
}

function Tab({ value, children }: { value: string; children: ReactNode }) {
  return (
    <Tabs.Tab
      value={value}
      className="font-sign text-ink border-ink/60 bg-paint hover:border-ink focus-visible:outline-cat data-active:border-ink data-active:bg-sign grid h-12 cursor-pointer place-items-center rounded-[3px] border-2 border-dashed text-lg leading-none transition-colors hover:border-solid focus-visible:outline-2 focus-visible:outline-offset-2 data-active:border-solid"
    >
      {children}
    </Tabs.Tab>
  );
}

// -----------------------------------------------------------------------------
// Gate and error
// -----------------------------------------------------------------------------

function AdminGate({ signedIn }: { signedIn: boolean }) {
  return (
    <div className="bg-paint text-ink flex min-h-dvh flex-col">
      <AdminHeader />
      <main className="flex flex-1 flex-col items-center px-3 pt-6 pb-10">
        <div className="border-ink bg-paint w-full max-w-104 rounded-[5px] border-2 p-1">
          <div className="border-ink/40 flex flex-col gap-4 rounded-[3px] border p-4 sm:p-5">
            <h1 className="font-sign text-xl leading-tight">
              {signedIn ? "Khusus admin" : "Masuk dulu"}
            </h1>
            <p className="text-pretty">
              {signedIn
                ? "Akun ini belum punya akses admin. Masuk dengan akun admin untuk menambah tempat dan kecamatan."
                : "Masuk dengan akun admin untuk menambah tempat dan kecamatan."}
            </p>
            {signedIn ? null : (
              <Link
                to="/masuk"
                search={{ kembali: "/admin" }}
                className={cn(signButtonClass, "grid h-11 place-items-center")}
              >
                Masuk
              </Link>
            )}
            <BackToMap />
          </div>
        </div>
      </main>
    </div>
  );
}

function AdminError() {
  return <ErrorBoard title="Halaman admin belum bisa dimuat." href="/admin" />;
}
