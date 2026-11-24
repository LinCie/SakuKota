import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";

import { AccountControl } from "~/features/auth/components/account-control";
import type { CategoryDetail, Place } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { ErrorBoard } from "~/shared/components/error-board";
import { Toaster } from "~/shared/components/toaster";
import { buttonVariants } from "~/shared/components/ui/button";
import { Wordmark } from "~/shared/components/wordmark";

import { MySuggestions } from "../components/my-suggestions";
import { SuggestionForm } from "../components/suggestion-form";
import type { Suggestion } from "../types";

type SuggestPageProps = {
  categories: CategoryDetail[];
  subDistricts: SubDistrict[];
  places: Place[];
  /** The user's suggestions, newest first. */
  suggestions: Suggestion[];
};

/** Where a signed-in user suggests a place and follows their suggestions. */
export function SuggestPage({
  categories,
  subDistricts,
  places,
  suggestions,
}: SuggestPageProps) {
  return (
    <Toaster>
      <div className="bg-background text-foreground flex min-h-dvh flex-col">
        <header className="bg-background border-b">
          <div className="flex items-center justify-between gap-3 px-3 py-2 md:px-5 md:py-3">
            <Wordmark />
            <AccountControl />
          </div>
        </header>

        <main className="mx-auto grid w-full max-w-5xl gap-10 px-3 py-6 md:px-5 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
          <section
            aria-labelledby="suggest-title"
            className="flex flex-col gap-6"
          >
            <div className="flex flex-col gap-3">
              <Link
                to="/"
                className={cn(
                  buttonVariants({ variant: "ghost" }),
                  "text-muted-foreground -ml-3 self-start",
                )}
              >
                <ArrowLeft aria-hidden="true" data-icon="inline-start" />
                Kembali ke peta
              </Link>
              <h1
                id="suggest-title"
                className="text-foreground text-2xl leading-tight font-semibold tracking-[-0.01em] text-balance"
              >
                Usulkan tempat
              </h1>
              <p className="text-muted-foreground max-w-prose text-sm text-pretty">
                Tahu tempat terjangkau yang belum ada di peta? Usulkan di sini.
                Admin meninjaunya dulu sebelum tempat itu tampil di peta.
              </p>
            </div>

            <SuggestionForm
              categories={categories}
              subDistricts={subDistricts}
              places={places}
            />
          </section>

          <section
            aria-labelledby="my-suggestions"
            className="bg-card ring-foreground/10 flex flex-col gap-3 rounded-xl p-5 ring-1 lg:sticky lg:top-6"
          >
            <h2
              id="my-suggestions"
              className="text-foreground text-base font-semibold"
            >
              Usulanku
            </h2>
            <MySuggestions
              suggestions={suggestions}
              categories={categories}
              subDistricts={subDistricts}
            />
          </section>
        </main>
      </div>
    </Toaster>
  );
}

/** Shown when the suggest page's data fails to load. */
export function SuggestError() {
  return (
    <ErrorBoard title="Halaman usulan belum bisa dimuat." href="/usulan" />
  );
}
