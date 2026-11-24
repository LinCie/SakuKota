---
name: SakuKota
description: A hyper-local survival map for young perantau in Jakarta, set soft and quiet on warm linen, with category colours borrowed from jajan pasar.
colors:
  background: "oklch(0.982 0.006 80)"
  card: "oklch(0.995 0.003 80)"
  foreground: "oklch(0.27 0.012 60)"
  muted: "oklch(0.955 0.009 80)"
  muted-foreground: "oklch(0.5 0.016 65)"
  accent: "oklch(0.945 0.014 85)"
  border: "oklch(0.905 0.01 75)"
  input: "oklch(0.85 0.014 75)"
  primary: "oklch(0.5 0.075 150)"
  primary-foreground: "oklch(0.985 0.006 80)"
  ring: "oklch(0.6 0.075 150)"
  destructive: "oklch(0.55 0.15 28)"
  sambal: "oklch(0.62 0.13 32)"
  sambal-soft: "oklch(0.94 0.024 40)"
  sambal-deep: "oklch(0.46 0.12 32)"
  ubi: "oklch(0.58 0.09 305)"
  ubi-soft: "oklch(0.94 0.02 310)"
  ubi-deep: "oklch(0.44 0.09 305)"
  telang: "oklch(0.58 0.09 255)"
  telang-soft: "oklch(0.94 0.018 245)"
  telang-deep: "oklch(0.44 0.09 255)"
  kunyit: "oklch(0.78 0.12 80)"
  kunyit-soft: "oklch(0.95 0.04 88)"
  kunyit-deep: "oklch(0.47 0.085 70)"
  pandan: "oklch(0.58 0.08 150)"
  pandan-soft: "oklch(0.94 0.022 145)"
  pandan-deep: "oklch(0.43 0.07 150)"
typography:
  wordmark:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "-0.02em"
  figure:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.75rem"
    fontWeight: 600
    lineHeight: 1.1
    letterSpacing: "-0.02em"
    fontFeature: "tnum"
  headline:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.01em"
  title:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 600
    lineHeight: 1.4
  body:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.55
  body-sm:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.45
  label:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: 1.43
  caption:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 500
    lineHeight: 1.33
rounded:
  md: "8px"
  lg: "10px"
  xl: "14px"
  2xl: "18px"
  3xl: "22px"
  full: "9999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "24px"
  control-sm: "36px"
  control: "40px"
  control-touch: "44px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-foreground}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "0 16px"
    height: "{spacing.control-touch}"
  button-secondary:
    backgroundColor: "{colors.muted}"
    textColor: "{colors.foreground}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "0 16px"
    height: "{spacing.control-touch}"
  button-outline:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "0 16px"
    height: "{spacing.control-touch}"
  button-ghost-hover:
    backgroundColor: "{colors.muted}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.full}"
  search-bar:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    typography: "{typography.body}"
    rounded: "{rounded.full}"
    height: "{spacing.control-touch}"
  select-kecamatan:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    typography: "{typography.title}"
    rounded: "{rounded.full}"
    padding: "0 12px 0 14px"
    height: "{spacing.control-touch}"
  category-filter-trigger:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "0 12px 0 16px"
    height: "{spacing.control-touch}"
  category-filter-trigger-active:
    backgroundColor: "{colors.pandan-soft}"
    textColor: "{colors.pandan-deep}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "0 12px 0 16px"
    height: "{spacing.control-touch}"
  category-tile:
    rounded: "{rounded.lg}"
    size: "32px"
  category-badge:
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: "2px 10px"
  map-pin:
    rounded: "{rounded.full}"
    size: "32px"
  map-pin-selected:
    backgroundColor: "{colors.foreground}"
    textColor: "{colors.card}"
    rounded: "{rounded.full}"
    size: "40px"
  card:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.xl}"
    padding: "20px"
  popover-menu:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.2xl}"
    padding: "6px"
  dialog:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.3xl}"
    width: "min(100% - 1.5rem, 34rem)"
  sheet-bottom:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.3xl}"
  list-row-hover:
    backgroundColor: "{colors.muted}"
    padding: "12px 20px"
  missing-slot:
    backgroundColor: "{colors.muted}"
    textColor: "{colors.muted-foreground}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.md}"
    padding: "2px 8px"
  input-field:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    typography: "{typography.body}"
    rounded: "{rounded.full}"
    padding: "0 14px"
    height: "{spacing.control-touch}"
  field-label:
    textColor: "{colors.foreground}"
    typography: "{typography.label}"
  field-error:
    textColor: "{colors.destructive}"
    typography: "{typography.body-sm}"
  form-alert:
    backgroundColor: "{colors.sambal-soft}"
    textColor: "{colors.sambal-deep}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.xl}"
    padding: "10px 14px"
  status-pending:
    backgroundColor: "{colors.muted}"
    textColor: "{colors.muted-foreground}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
  status-approved:
    backgroundColor: "{colors.pandan-soft}"
    textColor: "{colors.pandan-deep}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
  status-rejected:
    backgroundColor: "{colors.sambal-soft}"
    textColor: "{colors.sambal-deep}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
---

# Design System: SakuKota

## Overview

**Creative North Star: "Jajan Pasar on Linen"**

SakuKota is calm. The newcomer opens it late, tired, on a phone, and it should feel like a friend's tidy notebook, not a street shouting at them. Every surface is warm linen and soft white with hairline borders; the chrome stays neutral and quiet, and colour belongs only to what the visitor is looking for. That colour comes from the morning market's jajan pasar: sambal clay, ubi purple, telang blue, kunyit yellow, and pandan green, the soft, familiar tints of kue lapis, klepon, and nasi kuning. They mark categories and nothing else, so a glance at the map reads as "food here, clinic there" before any label is read.

Minimalism here means subtraction, not blankness: one typeface (Figtree), one accent (pandan), one elevation idea (only things floating over the map cast a shadow), and generous but working-app density. Corners are soft: controls are pills, containers are gently rounded. State changes by tint, not by weight: a selected filter takes its soft tint, a selected pin turns to ink, a hovered row warms to muted.

**Family, not twin.** SakuKota is a sibling of LinCie (`~/code/lincie`, "The Station House in Morning Light") and shares its temperament: warm low-saturation grounds instead of sterile white, tinted ink instead of black, quiet borders, flat-by-default surfaces, generous calm spacing, and accents kept rare. It is deliberately not LinCie's world:

- **Place and hour:** LinCie is a rural station at morning; SakuKota is a Jakarta kecamatan after work. Its warmth is the indoor warmth of linen and kitchen colours, not sunlit landscape.
- **Ink and ground:** LinCie writes in moss-green ink on yellowed sunlit paper; SakuKota writes in brown-charcoal ink on paler linen.
- **Accent:** LinCie's signal is terracotta with field green as landscape; SakuKota's one accent is a deeper, cooler pandan, and its terracotta-like sambal is a category, never an action colour.
- **Voice:** LinCie pairs a literary serif with a sans; SakuKota is one rounded sans (Figtree), because it is a tool read at a glance, not a story.
- **Form:** LinCie uses architectural rounded rectangles and keeps pills for dots; SakuKota's controls are pills, the shape of map chips under a thumb.
- **Colour system:** LinCie's colours are scenery; SakuKota's are data, the five jajan pasar tints that each mean a category.

This replaces the earlier "Warung Signboard" world (Bungee lettering, saturated enamel plates, 2px ink outlines, chili stripes). Product behaviour, layout, copy, and data rules are unchanged; only the visual language moved.

**Implementation status:** migration complete. Every surface (the map, list panel, place details, reviews, forms, auth pages, `/usulan`, dialogs, error pages, and the admin console) uses shadcn/ui and these tokens; the signboard tokens and Bungee are removed.

**Light only.** There is no dark theme: `app/app.css` declares `color-scheme: light`, has no `.dark` block and no `dark` custom variant, and shadcn's `dark:` classes stay inert.

**Key Characteristics:**

- Warm linen ground, soft-white cards, warm charcoal ink. No pure white, no pure black, no grey without warmth.
- One accent: pandan green for primary actions, focus, selection, and the active kecamatan.
- Five category tints, each with a soft fill, a mid solid, and a deep text tone. Colour means category.
- Figtree only, weights 400 to 700; hierarchy by size and weight, never by a second face.
- Pills for controls, 14px for cards, 18px for menus, 22px for dialogs and sheets.
- Hairline borders on resting surfaces; soft ambient shadows only on things floating over the map.
- Missing data is still drawn, as a quiet muted slot with explicit copy.

## Colors

A restrained strategy: warm neutrals plus one accent carry the interface; the category tints are a data palette, not decoration. All tokens are OKLCH (shadcn's format); hex values are approximations for design tools.

Tokens follow shadcn's convention in `app/app.css`: each value is a CSS variable in `:root` (`--background`, `--primary`, `--radius`, …), exposed to Tailwind by `@theme inline` (`--color-background: var(--background)`), with shadcn's `@layer base` block setting the default border and outline colours and the body's background and text. The category tints are added the same way as shadcn's "new tokens" (`--sambal-soft` in `:root`, `--color-sambal-soft: var(--sambal-soft)` in `@theme inline`), giving utilities like `bg-sambal-soft` and `text-sambal-deep`. shadcn's unused chart tokens may stay; the sidebar tokens map to linen, warm ink, pandan, pandan-soft, and the hairline.

### Primary

- **Pandan** (primary, `#426f4c`): primary buttons ("Kirim ulasan", submit buttons, "Coba lagi"), links, the active kecamatan outline, the selected item in the admin sidebar, text selection fill (as pandan-soft). Text on it is primary-foreground (5.5:1).
- **Pandan ring** (ring, `#5f8d68`): every keyboard focus ring (3.6:1 against linen).

### Neutral

- **Linen** (background, `#fbf9f5`): page, header, list column, auth and admin pages.
- **Soft white** (card, `#fefdfb`): cards, inputs, popovers, dialogs, the bottom sheet, map floaters.
- **Warm ink** (foreground, `#2b2521`): all primary text (14.3:1 on linen), the selected map pin.
- **Muted** (muted, `#f3f0ea`): secondary buttons, list-row hover, table headers, missing slots, skeletons.
- **Muted ink** (muted-foreground, `#6a625a`): addresses, hints, placeholders, timestamps, the list count (5.7:1 on linen, 5.3:1 on muted).
- **Warm hover** (accent, `#f1ece3`): menu item highlight and ghost-button hover (shadcn's `accent`).
- **Hairline** (border, `#e4dfd8`): dividers, card and panel borders, the header's bottom rule.
- **Field edge** (input, `#d3cdc4`): input, select, and outline-button borders; the scrollbar thumb.
- **Clay red** (destructive, `#b9473d`): field errors, destructive buttons, the "Ditolak" state (5.0:1 on linen).

### Category tints (jajan pasar)

Each tint has three tones: **soft** for fills behind text (chips, badges, list tiles, active filter), **solid** for map pins and icon-only marks (white icons pass 3:1 on every solid except kunyit), and **deep** for text and icons on the soft fill (6:1 or better).

| API `color` key | Tint       | Soft / Solid / Deep               | Theme                | Categories                                         |
| --------------- | ---------- | --------------------------------- | -------------------- | -------------------------------------------------- |
| `chili`         | **Sambal** | `#fae6df` / `#c86553` / `#8f392a` | Food and daily goods | Warung, Minimarket, Air & Gas                      |
| `plum`          | **Ubi**    | `#efe8f5` / `#876ca5` / `#5e457a` | Crafts               | Penjahit, Fotokopi                                 |
| `cat`           | **Telang** | `#e2edf7` / `#557caf` / `#2e5483` | Health and emergency | Klinik, Rumah Sakit, Puskesmas, Pos Polisi         |
| `sign`          | **Kunyit** | `#faeed1` / `#e0ae57` / `#79511e` | City services        | Laundry, Bank & ATM, Halte/Stasiun, Layanan Publik |
| `board`         | **Pandan** | `#e3efe2` / `#578861` / `#325b3b` | Care and community   | Apotek, Masjid, Ibadah Lain                        |

The API's `color` keys are unchanged; the app maps each key to its tint. Telang solid also marks the visitor's own location; kunyit solid fills earned review stars.

### Named Rules

**The Colour Means Category Rule.** Tints appear only where a category is meant (pins, tiles, badges, the active filter) plus the few status uses named here. Chrome is never tinted for decoration.

**The Kunyit Takes Deep Rule.** Kunyit is too light for white: kunyit pins carry a kunyit-deep icon (3.4:1), and kunyit text always sits as kunyit-deep on kunyit-soft.

**The Solid Carries No Text Rule.** Solid tints hold icons only. Any text on a category colour is the deep tone on the soft tone.

**The Warm Neutral Rule.** Every neutral carries a little warmth (hue 60–85). No `#fff`, `#000`, or cool grey anywhere, including shadows, which are tinted from warm ink.

## Typography

**Font:** Figtree Variable (with ui-sans-serif, system-ui). Bungee is retired.

**Character:** Figtree is round-shouldered and friendly without being cute, and it stays legible small on a phone at night. One family keeps the page quiet; weight and size do all the work.

### Hierarchy

- **Wordmark** (700, 1.25rem, −0.02em): "SakuKota" in warm ink with "Kota" in pandan. No frame.
- **Figure** (600, 1.75rem, −0.02em, tabular): the daily-spending amount in the kecamatan info.
- **Headline** (600, 1.25rem, 1.3, balanced): the place name in details, dialog titles, page titles (1.5rem on admin and `/usulan`).
- **Title** (600, 1rem): section headings ("Daftar tempat", "Ulasan", info sections), list place names, the kecamatan select value.
- **Body** (400, 1rem, 1.55): descriptions and kecamatan notes, capped at ~65ch with pretty wrapping.
- **Body small** (400, 0.875rem): addresses, hints, review comments, the list count.
- **Label** (500, 0.875rem): buttons, field labels, filter labels, detail-row labels.
- **Caption** (500, 0.75rem): badges, timestamps, map attribution.

### Named Rules

**The One Family Rule.** Only Figtree. Emphasis is weight 600–700, never italics for emphasis, never a display face.

**The Tabular Money Rule.** Prices, counts, ratings, and the daily-spending figure use tabular numerals.

## Layout

Unchanged from the product's wireframe: a single-screen app exactly the viewport tall that never scrolls; the map owns everything below the header. The list and place details live in a toggleable panel, never below the map.

- **Desktop (768px+):** the panel is a 24rem column on the right, open by default, separated from the map by a hairline. Only the panel scrolls. A floating "Sembunyikan daftar" button at the map's top right hides it.
- **1024px+:** the header is one 64px row: the wordmark, then the search bar (up to 48rem) 32px after it, and the account controls at the far right. The count sits in the list panel's heading and on the list toggle.
- **640–1023px:** two rows: wordmark left and account controls right, then the search bar across the full width.
- **Phones (<640px):** two rows above the map (~120px). Row one: a smaller wordmark, the kecamatan select as its own pill filling the middle, an icon-only account button. Row two: the search bar holding only search and an icon-only category filter. "Info kecamatan" moves onto the map as a floating pill at the top right. A floating "Daftar tempat" pill with the count sits at the bottom left and opens a bottom sheet (up to 60% of the viewport). Selecting a place swaps the sheet to its details; closing returns to the list if it was open, else to the bare map. The map pans so the selected pin stays above the sheet.
- **Auth pages:** the wordmark where the map header puts it, then one card up to 26rem wide in a centred column, top-aligned on phones and anchored 12% from the top from 768px (never vertically centred, so it holds still between pages and errors). "Kembali ke peta" sits 12px under the card.
- **`/usulan`:** a scrolling page under the site header: a column up to 64rem with the page header (back link, headline, lede) and the form; from 1024px a sticky 20rem "Usulanku" card on the right.

Spacing runs on a 4px grid: 12px side padding on phones, 20px from 768px; 8px between sibling controls; 16px inside rows and cards; 24px between form sections. Touch targets are 44px on phones and 40px from 768px; dense admin controls may use 36px.

## Elevation & Depth

Resting surfaces are flat: cards and panels separate from linen by a 1px hairline (`border`) or shadcn's `ring-1 ring-foreground/10`, never both a border and a shadow. Only things floating above the map or the page cast a shadow, and every shadow is soft, offset downward, and tinted from warm ink.

### Shadow Vocabulary

- **Pin** (`0 2px 6px oklch(0.27 0.012 60 / 0.25)`): each map pin.
- **Float** (`0 4px 16px oklch(0.27 0.012 60 / 0.10), 0 1px 3px oklch(0.27 0.012 60 / 0.08)`): zoom, "Lokasi saya", the list toggle, "Info kecamatan" on phones, the empty-state card, menus and popovers.
- **Sheet** (`0 -8px 32px oklch(0.27 0.012 60 / 0.12)`): the phone bottom sheet, cast upward.
- **Dialog** (`0 24px 64px oklch(0.27 0.012 60 / 0.18)`): dialogs and the admin form sheet, over a `foreground/30` backdrop. No backdrop blur.

### Named Rules

**The Only-Floaters-Cast Rule.** A shadow means "lifted above the map". On desktop the details panel drops the sheet shadow because it no longer floats.

## Shapes

Soft and consistent, from shadcn's radius scale with `--radius: 0.625rem`:

- **Pill (full):** every control: buttons, inputs, the kecamatan select, the category filter, badges, status chips, floating map buttons.
- **10px (lg):** category tiles in the list, image frames inside cards, table and menu inner elements.
- **14px (xl):** cards (place details sections, auth card, "Usulanku", admin cards), the media gallery frame, form alerts, menu items.
- **18px (2xl):** menus and popovers.
- **22px (3xl):** dialogs, and the bottom sheet's top corners only.
- **Circle:** map pins, the visitor's location dot, avatar initials, gallery dots.

Borders are 1px. Dashed survives only for the missing slot's optional outline; dotted leaders are retired.

## Components

Built on **shadcn/ui** (style `base-maia`, Base UI primitives, Lucide icons). Generated primitives live in `app/shared/components/ui/`; the project owns that code and edits it to this spec. App-specific compositions (map pins, the place list, the info dialog) live in their feature folders and compose those primitives.

### Buttons (`ui/button`)

- **Shape:** pill, 44px on phones, 40px from 768px (`size="lg"`/default adjusted); 36px (`sm`) only in dense admin tables.
- **Primary:** pandan with primary-foreground label (Figtree 500). Hover darkens toward pandan-deep (`hover:bg-primary/90`).
- **Secondary:** muted fill, ink label; hover to accent.
- **Outline:** soft-white fill, `input` border, ink label: the category filter, account links, and other resting secondary actions.
- **Floating:** soft-white fill with the float shadow and no border; hover to muted. Everything that floats over the map: zoom, "Lokasi saya", the list toggle, and "Info kecamatan" on phones.
- **Ghost:** no fill; muted fill on hover. Close, back, clear, and icon buttons.
- **Destructive:** shadcn's soft destructive (`destructive/10` fill, clay-red label) for delete and reject.
- **Focus:** `ring-3 ring-ring/50` plus `border-ring` (shadcn default). **Disabled:** 50% opacity. **Pending:** disabled with a spinner and "Memproses…".
- **Icons:** Lucide at the default 2px stroke, 16px in buttons, 20px standalone.

### Category filter (`ui/dropdown-menu`)

- **Trigger:** the search bar's last segment (at least 11rem), reading "Semua kategori", the category's label, or "N kategori", with a trailing chevron; hover fills it muted. On phones a 44px icon-only segment with a filter icon and the count when filtering, its label kept for screen readers.
- **Active:** the segment fills pandan-soft with pandan-deep text, an end cap inside the bar.
- **Menu:** a popover (18px, float shadow) at least as wide as the trigger, scrolling when taller than the screen: "Semua kategori" (clears; disabled when nothing is filtered), a separator, then one `DropdownMenuCheckboxItem` per category with its tile, label, and a check. The menu stays open while toggling.

### Category tile and badge

- **Tile:** a 32px rounded-lg square in the category's soft tint holding its Lucide icon (16px) in the deep tone. Used in list rows and the filter menu.
- **Badge (`CategoryBadge`):** pill, soft tint fill, deep-tone 14px icon and label in caption type. Heads the place details and labels categories in admin tables and suggestion lists.
- Letter codes (W, J, K…) from the API are no longer drawn; the icon and label identify the category.

### Cards / Containers (`ui/card`)

- **Place details:** no outer frame. A header row (a ghost "Daftar tempat" back button on desktop; the grab handle and a ghost close button on phones) over a hairline, then badge, headline, the media gallery (4:3, 14px), body description, then a detail list: label (muted ink, label type) left, value (ink, title weight) right, rows divided by hairlines. Rows run rating (a kunyit star, the average, the count in muted ink, or a missing slot "Belum ada ulasan"), opening hours, price, then the category's custom fields in field order by data type: numbers, yes/no (16px check or x icon in muted ink), choices, ranges ("12–24"), dates ("2 Maret 2015") as rows; free text and the address as stacked blocks under a hairline; a missing value is a missing slot "Data tidak tersedia". A small ghost "Laporkan" in muted ink sits right-aligned under the list for signed-in users, then the reviews.
- **Kecamatan info (`ui/dialog`):** 22px dialog, dialog shadow, a sticky header with the headline and a ghost close button, then sections separated by hairlines, each headed by a 20px muted icon and a title: photos (gallery 8:5), flood history, safety, and estimated daily spending (the figure at 1.5rem in ink, semibold and tabular, with "per hari, perkiraan" in muted body-small). On phones it is the same centred dialog, 12px from the edges, scrolling under its sticky header.
- **Empty state (`ui/empty`):** a soft-white card with float shadow centred over a `background/70` scrim on the map: one message and one primary button.

### Inputs / Fields (`ui/input`, `ui/select`, `ui/native-select`, `ui/textarea`)

- **Search bar (map header):** one soft-white pill with an `input` border, 44px (40px from 768px), split by 20px hairline dividers into segments with no borders of their own: the kecamatan select and its info button, search, and the category filter. Segments hover to muted as full-height fills, rounded only at the bar's own ends and square where they meet another segment; the bar takes the focus ring while search has focus, the other segments draw their own.
- **Kecamatan select:** a segment with a leading map-pin icon in pandan, the value in title weight, and a chevron; options in a popover. A 36px ghost info button in pandan follows it ("Info kecamatan", which opens the info dialog). On phones the select stands alone as a pill in the top row and the info moves onto the map.
- **Search:** a segment with a leading search icon in muted ink and a trailing ghost clear button; the native cancel control is hidden.
- **Form field:** label (label type) above a pill input with `input` border; textareas use 14px corners. Focus is the ring. An optional hint in muted body-small sits under it.
- **Invalid:** after submit the border turns destructive with a soft destructive ring, and the message (body-small, clay red) replaces the hint; editing clears it.
- **Form select (`ui/native-select`):** form pickers stay native selects (the best picker on a phone), drawn as the same pill field with a muted chevron; the empty choice reads in muted ink. Admin toolbars use the 36px `sm` size.
- **Choice toggles:** yes/no fields are a row of outline pills (at least 64px wide); the picked one fills pandan with white text.
- **Form alert:** sambal-soft fill, sambal-deep text and alert icon, 14px corners, at the top of the form, assertive.

### Navigation

- **Site header:** the map, `/usulan`, and `/profil` share one: linen, a hairline bottom border, no fill colour, the wordmark left (no tagline) and the account controls right; the map adds its search bar.
- **Count:** the list heading and the list toggle show it; a hidden polite live region announces the full sentence ("N tempat di Kec. X untuk “q”").
- **Page header:** the title block of admin pages and `/usulan`: an optional ghost back link, the headline with the page's actions beside its first line (dropping under it when they do not fit), and a muted body-small description under both.
- **Account controls:** signed out, an outline pill "Masuk" with a user icon linking to the sign-in page with the current map address. Signed in: "Usulkan" (outline pill, pin-plus icon) except on `/usulan` and in admin, "Admin" (outline pill, shield icon) for admins outside admin, the first name in title weight truncated at 10ch, and a ghost "Keluar" with a log-out icon. On phones a 44px icon-only pill: signed out it links to sign-in; signed in it opens a dropdown menu headed by the full name with "Usulkan", "Admin" for admins, and "Keluar".
- **List panel:** from 768px a linen column with a 1px hairline on its left and no shadow; on phones a soft-white bottom sheet with 22px top corners and the sheet shadow. Its sticky heading reads "Daftar tempat" (title) with the count and, on phones, a ghost close button, over a hairline.
- **Place list:** rows divided by hairlines: category tile, name (title), truncated address (muted body-small), right-aligned tabular price. Hover and focus fill muted, with an inset focus ring. Loading more is a muted line; a failed page is a pandan underlined "Gagal memuat. Coba lagi".

### Map Pin (signature)

A place is a 32px circle in its category's solid tint with a white 16px Lucide icon (kunyit-deep on kunyit), ringed by 2px soft white, with the pin shadow, anchored at its centre. Hover scales it to 110%. Selected becomes a 40px warm-ink circle with a soft-white icon and a 3px soft-white ring, raised above other pins. Keyboard focus draws the pandan ring. Selection is marked with `data-selected="true"`, the value shadcn's `data-selected` variant matches.

**Settling motion:** when pins appear they fade in and scale from 0.6 to 1 over 400ms with `cubic-bezier(0.16, 1, 0.3, 1)`, staggered west to east in 30ms steps up to 300ms. Changing kecamatan flies the map to the new bounds over 0.9s. Reduced motion jumps and skips the settle.

**Map canvas:** Esri Light Gray base and reference tiles (no POI icons) warmed with a 20% sepia filter over a muted backing so the tiles sit on linen. The default Leaflet zoom control is replaced by a floating soft-white pill pair (float shadow, hairline divider, 44px buttons, 40px from 768px) at the top left; attribution sits on `card/90` in caption type with pandan links, with a rounded top-left corner. Neighbour tooltips are soft-white pills with the float shadow and no arrow. Minimum zoom 13, no clustering.

**Kecamatan outlines:** the kecamatan on show is outlined in 2px pandan over a 6% pandan wash and lets taps through. Neighbours are dashed (6px on, 6px off) 1.5px outlines in muted ink at 50%, no fill; hovering one washes it in 10% pandan and names it in a tooltip ("Kec. Tebet"); clicking picks it. Outlines sit under the pins. A kecamatan without a boundary has no outline.

**Lokasi saya:** a floating circle button (44px; 40px from 768px, float shadow) at the bottom right with a locate icon that spins while locating and turns telang once found. The visitor is a 16px telang dot with a 3px soft-white ring over a slow telang/20 pulse, inside a faint telang accuracy circle. A failure shows a short warm-ink toast-like note above the button (14px corners, soft-white text) that clears after 6 seconds.

**Location picker (forms):** the same canvas in a 16rem frame with a hairline border and 14px corners, with the zoom pair. The place being placed is a 40px warm-ink pin with its category icon (a question mark before a category is picked); places already there are quiet 24px pins at 60% opacity.

### Auth (Masuk / Daftar)

Linen page, no coloured wall. The wordmark with its tagline ("Peta bertahan hidup untuk perantau di Jakarta", only here) links back to the map.

- **Card:** soft white, hairline ring, 14px corners, 20px padding (24px from 640px), flat.
- **Switch:** a segmented pill pair (shadcn Tabs look, built from links): the current page is the raised soft-white segment with a subtle shadow and is the page's `<h1>`; the other is a muted-ink link segment on muted. Switching swaps only the form.
- **Form:** follows the switch 20px below with fields at a 16px gap and a full-width primary submit.
- **Return link:** a ghost "Kembali ke peta" with a back arrow under the card. It and the wordmark return to the exact map address the visitor came from.

### Missing Slot

A quiet, explicit gap: muted fill, muted-ink body-small text, 8px corners, 2px × 8px padding, with copy like "Jam buka tidak tersedia" or "Data tidak tersedia". A missing image is a muted frame at the image's own ratio (4:3 place, 8:5 kecamatan photo or area map) with an image-off icon and the same copy.

### Media Gallery

One photo or video at a time in a 14px frame on muted; videos use the browser's controls. With more than one: 36px floating circle arrows at each side (float shadow) and a centred row of 8px dots under the frame (warm ink for current, `input` for the rest). Slides move over 300ms ease-out, not at all under reduced motion. The gallery takes focus with the ring; arrow keys move it.

### Reviews

Under the details, headed "Ulasan" (title). Each review: author (title weight), five 16px stars on the right (earned kunyit-filled, the rest `input` outlined), the comment, and the date in caption muted ink, divided by hairlines. None is a missing slot "Belum ada ulasan". Deleting is a small ghost "Hapus" in clay red; reporting is the ghost "Laporkan".

- **Signed out:** a secondary (muted) pill "Masuk untuk menulis ulasan", since an outline would vanish on the soft-white sheet, that returns to the same place after sign-in.
- **Signed in:** five 44px star targets with 28px stars (kunyit-soft hover; unpicked in `input`, destructive outline after a failed submit), an optional textarea, and a primary "Kirim ulasan". Once reviewed, a muted line replaces the form.

### Suggest Page (`/usulan`)

The map's header, then the layout described above. "Usulanku" is a card listing the user's suggestions divided by hairlines, each with its status badge and, when rejected, the reason.

**Status badge:** pill, caption type. "Menunggu" muted with muted ink; "Disetujui" pandan-soft with pandan-deep; "Ditolak" sambal-soft with sambal-deep.

### Admin Console (`/admin`)

The same world at higher density, assembled from shadcn's admin primitives.

- **Frame:** a 15rem linen sidebar from 768px, with a hairline on its right, holding the wordmark with an "Admin" outline badge, 36px nav items (label type, 16px icon, 10px corners), and at its foot, over a hairline, "Kembali ke peta" (a muted nav item) and the account controls. On phones a top row holds the wordmark, badge, and account, over the same items as a scrolling strip; the wordmark is the way back. The active item is pandan-soft with pandan-deep text. There is no top bar.
- **Page:** linen, 16px padding (24px from 768px), opening with the page header.
- **Search:** the pill search field at 36px, up to 24rem wide, with the row count in muted beside it.
- **Tables (`DataTable`):** inside a card; header row on muted with muted-ink caption-weight labels; rows divided by hairlines, hovering to muted (a column pinned for sideways scrolling on phones follows the row). Statuses are soft badges: open ones muted, resolved ones in their outcome's tint (pandan for approved or dismissed, sambal for rejected or removed).
- **Buttons:** 36px pills (`sm`); primary, outline, ghost, and soft destructive as above.
- **Form drawer (Base UI drawer, right):** up to 32rem, soft white with the dialog shadow over `foreground/30`, swipeable to close; title and description at the top with a ghost close, the form scrolling, outline "Batal" and the primary action in a footer.
- **Confirm (`ui/alert-dialog`):** also used on the map for deleting a review: a 22px dialog, title and muted description, an outline "Batal" and a soft destructive (or primary) action; on phones centred with stacked full-width buttons.
- **Detail pages:** a ghost back link, the headline with outline, primary, and soft destructive actions, then a grid of cards with label and value pairs (muted caption labels over ink values).
- **Row actions (`ui/dropdown-menu`):** a ghost 36px "more" button opening the soft popover menu; destructive items in clay.
- **Pagination:** the muted page count with outline 36px previous and next buttons.
- **Skeletons:** muted pulses in the shape of the page.
- **Toasts (`ui/toast`):** soft-white popovers with float shadow, used for finished writes here and on `/usulan`.
- **Gate:** non-admins see a small card: "Masuk dulu" with a sign-in button when signed out, "Khusus admin" when signed in, and a link back to the map.

### Error page

When the map or `/usulan` cannot load, the page is linen with one centred card: the wordmark, the failure in semibold, a muted line about the server, and a primary "Coba lagi". An admin page that fails shows the same message as a card inside the console, with an outline "Coba lagi".

### Browser surfaces

Text selection is pandan-soft with warm ink; the caret is pandan; scrollbars use an `input` thumb on a transparent track; focus rings are always the pandan ring.

### Motion

One ease for everything that enters: `cubic-bezier(0.16, 1, 0.3, 1)`. The bottom sheet slides up over 300ms; dialogs and menus use shadcn's fade and 95% zoom over 150–200ms; the auth card rises 8px and fades in over 400ms; hovers transition colour at 150ms. The map pin settle is the one authored moment. Everything is gated on `prefers-reduced-motion: no-preference`.

## Do's and Don'ts

### Do:

- **Do** build controls from `app/shared/components/ui` (shadcn, Base UI, Lucide) and edit those files to match this spec rather than overriding them at every call site.
- **Do** style with Tailwind utilities and the shadcn tokens (`bg-background`, `text-muted-foreground`, `border-border`, `bg-sambal-soft`, and so on). The only stylesheet code is shadcn's own theme blocks in `app/app.css` (`:root` variables, `@theme inline`, `@layer base`, and its imports).
- **Do** keep colour for categories: soft fill behind deep text, solid only for pins and icons.
- **Do** keep every neutral warm and every shadow tinted from warm ink.
- **Do** draw missing data as a muted missing slot with explicit Bahasa Indonesia copy.
- **Do** keep touch targets at 44px on phones and 40px from 768px.
- **Do** gate every animation on reduced-motion preference.

### Don't:

- **Don't** bring back the signboard vocabulary: no Bungee, 2px ink outlines, inset frames, chili stripes, dotted leaders, or letter-code plates.
- **Don't** put white or ink text on a solid category tint, or any text on kunyit solid.
- **Don't** pair a border with a shadow on the same surface, or give resting cards a shadow.
- **Don't** use pure white, pure black, cool greys, gradients, glass, or backdrop blur.
- **Don't** add a second typeface or use italics for emphasis.
- **Don't** borrow LinCie's clothes: no serif display, moss ink, terracotta actions, or landscape scenery. Share its temperament, not its look.
- **Don't** add map markers or tile styles that compete with the pins; keep the canvas quiet and POI-free.
