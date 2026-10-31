import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";

import { AccountControl } from "~/features/auth/components/account-control";
import type { CategoryDetail, Place } from "~/features/places/types";
import type { SubDistrict } from "~/features/subdistricts/types";
import { cn } from "~/shared/cn";
import { menuButtonClass } from "~/shared/components/menu-board";
import { Toaster } from "~/shared/components/toaster";
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
      <div className="bg-paint text-ink flex min-h-dvh flex-col">
        <header className="bg-board text-paint">
          <div className="flex items-center justify-between gap-3 px-3 py-3 md:px-5">
            <Wordmark />
            <AccountControl />
          </div>
          <div aria-hidden="true" className="bg-chili h-1.5" />
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
                  menuButtonClass,
                  "inline-flex h-10 items-center gap-2 self-start rounded-[3px] px-2 text-sm font-semibold",
                )}
              >
                <ArrowLeft
                  aria-hidden="true"
                  className="size-4"
                  strokeWidth={2.5}
                />
                Kembali ke peta
              </Link>
              <h1
                id="suggest-title"
                className="font-sign text-ink text-2xl leading-tight text-balance"
              >
                Usulkan tempat
              </h1>
              <p className="text-ink-soft max-w-prose text-sm text-pretty">
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
            className="border-ink flex flex-col gap-3 rounded-[5px] border-2 p-1 lg:sticky lg:top-6"
          >
            <div className="border-ink/40 flex flex-col gap-3 rounded-[3px] border p-4">
              <h2 id="my-suggestions" className="font-sign text-ink text-base">
                Usulanku
              </h2>
              <MySuggestions
                suggestions={suggestions}
                categories={categories}
                subDistricts={subDistricts}
              />
            </div>
          </section>
        </main>
      </div>
    </Toaster>
  );
}
