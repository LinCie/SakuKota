---
name: Rantau
description: A hyper-local survival map for young perantau in Jakarta, lettered like the street's own warung signs.
colors:
  board: "#1e6b3a"
  board-deep: "#14502a"
  sign: "#f2c200"
  sign-soft: "#fbe58a"
  chili: "#c8233a"
  cat: "#1f4e9c"
  plum: "#6a3d9a"
  paint: "#fff9ec"
  paint-deep: "#f3ead3"
  ink: "#1a1a17"
  ink-soft: "#55524a"
typography:
  display:
    fontFamily: "Bungee, Figtree Variable, ui-sans-serif, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 400
    lineHeight: 1
    letterSpacing: "0.025em"
  figure:
    fontFamily: "Bungee, Figtree Variable, ui-sans-serif, sans-serif"
    fontSize: "2.25rem"
    fontWeight: 400
    lineHeight: 1
    fontFeature: "tnum"
  headline:
    fontFamily: "Bungee, Figtree Variable, ui-sans-serif, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 400
    lineHeight: 1.25
  title:
    fontFamily: "Bungee, Figtree Variable, ui-sans-serif, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.5
  code:
    fontFamily: "Bungee, Figtree Variable, ui-sans-serif, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1
  body:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.5
  body-strong:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 600
    lineHeight: 1.5
  label:
    fontFamily: "Figtree Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 700
    lineHeight: 1.43
rounded:
  code-inner: "2px"
  plate: "3px"
  frame: "5px"
  dialog: "6px"
  sheet: "10px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  control-sm: "40px"
  control: "44px"
components:
  header-board:
    backgroundColor: "{colors.board}"
    textColor: "{colors.paint}"
    padding: "12px 20px"
  wordmark-plate:
    backgroundColor: "{colors.board-deep}"
    textColor: "{colors.sign}"
    typography: "{typography.display}"
    rounded: "{rounded.frame}"
    padding: "4px"
  button-sign:
    backgroundColor: "{colors.sign}"
    textColor: "{colors.ink}"
    typography: "{typography.code}"
    rounded: "{rounded.plate}"
    padding: "0 16px"
    height: "{spacing.control}"
  button-sign-hover:
    backgroundColor: "{colors.sign-soft}"
    textColor: "{colors.ink}"
  button-quiet-hover:
    backgroundColor: "{colors.sign-soft}"
    textColor: "{colors.ink}"
    rounded: "{rounded.plate}"
  select-kecamatan:
    backgroundColor: "{colors.board-deep}"
    textColor: "{colors.paint}"
    typography: "{typography.body-strong}"
    rounded: "{rounded.plate}"
    padding: "0 40px 0 12px"
    height: "{spacing.control}"
  input-search:
    backgroundColor: "{colors.paint}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.plate}"
    padding: "0 44px 0 40px"
    height: "{spacing.control}"
  filter-plate-inactive:
    backgroundColor: "{colors.paint}"
    typography: "{typography.label}"
    rounded: "{rounded.plate}"
    padding: "0 12px 0 4px"
    height: "{spacing.control-sm}"
  filter-plate-all-active:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paint}"
    typography: "{typography.label}"
    rounded: "{rounded.plate}"
    padding: "0 12px"
    height: "{spacing.control-sm}"
  filter-plate-warung-active:
    backgroundColor: "{colors.chili}"
    textColor: "{colors.paint}"
    typography: "{typography.label}"
    rounded: "{rounded.plate}"
    height: "{spacing.control-sm}"
  code-plate:
    typography: "{typography.code}"
    rounded: "{rounded.plate}"
    size: "28px"
  map-sign:
    typography: "{typography.code}"
    rounded: "{rounded.plate}"
    size: "36px"
  map-sign-selected:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.sign}"
    rounded: "{rounded.plate}"
    size: "36px"
  menu-board:
    backgroundColor: "{colors.paint}"
    textColor: "{colors.ink}"
    rounded: "{rounded.frame}"
    padding: "4px"
  notice-board-dialog:
    backgroundColor: "{colors.paint}"
    textColor: "{colors.ink}"
    rounded: "{rounded.dialog}"
    width: "min(100% - 1.5rem, 34rem)"
  notice-board-header:
    backgroundColor: "{colors.board}"
    textColor: "{colors.sign}"
    typography: "{typography.headline}"
    padding: "6px"
  missing-slot:
    textColor: "{colors.ink-soft}"
    typography: "{typography.label}"
    rounded: "{rounded.plate}"
    padding: "2px 8px"
  list-row-hover:
    backgroundColor: "{colors.sign-soft}"
    padding: "12px 20px"
  auth-wall:
    backgroundColor: "{colors.board}"
    textColor: "{colors.paint}"
    padding: "12px"
  auth-board:
    backgroundColor: "{colors.paint}"
    textColor: "{colors.ink}"
    rounded: "{rounded.frame}"
    padding: "4px"
    width: "min(100%, 26rem)"
  auth-plate-active:
    backgroundColor: "{colors.sign}"
    textColor: "{colors.ink}"
    rounded: "{rounded.plate}"
    height: "48px"
  auth-plate-inactive:
    backgroundColor: "{colors.paint}"
    textColor: "{colors.ink}"
    rounded: "{rounded.plate}"
    height: "48px"
  input-field:
    backgroundColor: "{colors.paint}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.plate}"
    padding: "0 12px"
    height: "{spacing.control}"
  field-label:
    textColor: "{colors.ink}"
    typography: "{typography.label}"
  field-error:
    textColor: "{colors.chili}"
    typography: "{typography.label}"
  form-alert:
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.plate}"
    padding: "10px 12px"
  button-submit:
    backgroundColor: "{colors.sign}"
    textColor: "{colors.ink}"
    typography: "{typography.code}"
    rounded: "{rounded.plate}"
    padding: "0 16px"
    height: "{spacing.control}"
    width: "100%"
  button-submit-pending:
    backgroundColor: "{colors.sign-soft}"
    textColor: "{colors.ink}"
  account-plate:
    textColor: "{colors.paint}"
    typography: "{typography.label}"
    rounded: "{rounded.plate}"
    padding: "0 12px"
    height: "{spacing.control}"
  account-plate-hover:
    backgroundColor: "{colors.board-deep}"
    textColor: "{colors.paint}"
---

# Design System: Rantau

## Overview

**Creative North Star: "The Warung Signboard"**

Rantau is lettered the way a Jakarta street is lettered: hand-painted warung and kaki-lima signage. Every surface is a flat painted panel (warung-green board, sign yellow, chili red, painted blue, plum, paint white, ink) with no gradients and no glossy app chrome. Components are signboards: rectangular plates with a thin painted inset border, category codes (W, J, K, L, A) lettered in Bungee, and pins that are miniature signs planted on a post. The map is a quiet, slightly warmed street canvas so the painted signs are the only loud things on it.

Density is working-app density, not marketing density. A green header board carries the wordmark, kecamatan picker, the yellow "Lihat info" plate, and search; a paint-white strip of category plates and a live count sits under it; the map owns the rest. Place details read as a menu board with dotted leaders; kecamatan facts are a notice board. Missing data is never hidden: it is drawn as an empty dashed slot. State changes by the mark itself: an active plate is solid paint, an inactive plate is a dashed outline, a selected pin inverts to ink with a yellow code.

The world explicitly refuses the super-app default of white cards, pill chips, and blue droplet pins. Tailwind CSS v4 utilities are the only styling mechanism in `web/`; tokens live in `app/app.css` `@theme` and no raw CSS or inline styles are written.

**Key Characteristics:**
- Flat painted colour fields; no gradients anywhere.
- Bungee for every painted letter (wordmark, plate codes, sign headings, prices and counts); Figtree for all reading text.
- Tight, near-square corners (2–6px); plates, not pills.
- 2px ink borders as the universal outline; dashed for "off" or "missing", dotted for menu-board leaders.
- State by mark: solid = on, dashed outline = off, ink inversion = selected.
- A 6px chili stripe closes every green board (header and notice board).
- Soft, ambient shadows only on things that float over the map.

## Colors

A signwriter's tin set: a few saturated, flat enamel colours over warm paint white, with ink for lettering and outlines.

### Primary
- **Warung Board Green** (board): the header board, notice-board header, the Apotek category plate, and info-section icons. The ground colour of the whole world.
- **Deep Board Green** (board-deep): recessed areas on the board: the wordmark frame, the kecamatan select field, hover on board-coloured buttons.

### Secondary
- **Sign Yellow** (sign): the lettering colour on green (wordmark, notice-board title, header inset borders), the primary action plate ("Lihat info", empty-state actions, "Coba lagi"), the Laundry category plate, the code on a selected pin, and focus borders on the header's fields. Also the text-selection colour.
- **Faded Sign Yellow** (sign-soft): the hover/focus fill for sign-yellow plates, quiet buttons, zoom buttons, and list rows (list rows at 60% strength).

### Tertiary
- **Chili Red** (chili): the Warung category, the 6px stripe under every green board, the daily-spending figure, and the text caret.
- **Painted Blue** (cat): the Klinik category, focus outlines on paint-white surfaces, and links in the map attribution.
- **Tailor Plum** (plum): the Penjahit category only.

### Neutral
- **Paint White** (paint): page, strips, list column, menu board, dialog body, search field. Text on every saturated plate except yellow.
- **Deep Paint** (paint-deep): the map's backing colour before tiles load, image backgrounds, and hover on the search clear button.
- **Ink** (ink): all primary text, every 2px outline, pin posts, the active "Semua" plate, the selected-pin inversion, and text on yellow.
- **Soft Ink** (ink-soft): secondary text (addresses, count sentence, leader labels, placeholder text), the missing-slot text, and the scrollbar thumb. Ink at reduced opacity (20–60%) supplies dashed dividers, inactive outlines, and the modal backdrop (60%).

### Named Rules
**The One Hue Per Category Rule.** Each category owns exactly one paint: Warung chili (W), Penjahit plum (J), Klinik blue (K), Laundry yellow (L), Apotek green (A). The same hue marks its filter plate, map sign, list code plate, and badge; unknown categories fall back to an ink plate with their initial.

**The Yellow Takes Ink Rule.** Sign yellow is never lettered in paint white. Yellow plates carry ink text and an ink border; the inactive Laundry plate is outlined in ink at 60%, not yellow, because yellow on paint white is illegible.

**The Flat Enamel Rule.** Colour fields are solid. No gradients, glass, or tinted overlays beyond ink-at-opacity dividers and the paint-at-70% scrim behind the empty state.

## Typography

**Display Font:** Bungee (with Figtree Variable, ui-sans-serif)
**Body Font:** Figtree Variable (with ui-sans-serif, system-ui)

**Character:** Bungee is the signwriter's brush: blocky, all-caps by design, used only where a sign would be painted. Figtree is the handwritten price list's clear, friendly counterpart for anything read at length.

### Hierarchy
- **Display** (Bungee 400, 1.5rem, line-height 1, tracking 0.025em): the RANTAU wordmark, yellow inside its inset frame.
- **Figure** (Bungee 400, 2.25rem, line-height 1, tabular numerals): the daily-spending amount on the notice board, in chili.
- **Headline** (Bungee 400, 1.5rem, line-height 1.25, balanced wrapping): the place name on the menu board; the notice-board title (1.25rem on phones, 1.5rem from 640px).
- **Title** (Bungee 400, 1rem): the "Daftar tempat" column heading and the live result count number.
- **Code** (Bungee 400, 0.875rem, line-height 1): category codes on plates and pins (1rem on map signs, 0.75rem in badges), and the text of yellow action plates.
- **Body** (Figtree 400, 1rem, line-height 1.5): descriptions and kecamatan notes, capped at prose width (~65ch) with pretty wrapping.
- **Body Strong** (Figtree 600, 1rem): place names in the list, leader-row values, the select value.
- **Label** (Figtree 600–700, 0.875rem): filter-plate labels, info-section headings, leader labels, addresses, prices in the list, the count sentence.

### Named Rules
**The Painted Letter Rule.** Bungee appears only where a sign painter would letter: wordmark, codes, sign headings, action plates, and money/count figures. Body copy, labels, and form values are always Figtree.

**The Tabular Money Rule.** Prices, counts, and the daily spending figure use tabular numerals so columns of Rupiah line up.

## Layout

A single-screen app. On desktop (768px and up) the page is exactly the viewport height: header board, category strip, then a 2:1 grid with the map on the left (2fr) and the menu-board column on the right (1fr), separated by a 2px ink rule; only the right column scrolls. From 1024px the header collapses onto one row (wordmark and tagline left, picker, "Lihat info", and search right, search capped at ~28rem) and the category strip puts plates left and the count right.

The header's account plate is the last item of that one-row header from 1024px. Below 1024px the header is a two-column grid: the wordmark cluster on the left and the account plate pinned to the right of the same row, with the picker, "Lihat info", and search spanning both columns beneath. The tagline is hidden below 640px to make room for the account plate.

The auth pages (`/masuk`, `/daftar`) are a separate full-viewport layout: the wordmark cluster sits where it does on the map header (12px padding, 20px from 768px), and a single board up to 26rem wide sits in a centred column under it. On phones the board is top-aligned just under the wordmark; from 768px it is anchored at a fixed offset from the top (12% of the dynamic viewport), never vertically centred, so the plates hold still across pages and error states. The "Kembali ke peta" link sits 12px under the board.

On phones the page stacks and scrolls: header controls wrap (picker and "Lihat info" share a row, search below), category plates wrap onto two rows with the count under them, the map takes 58% of the dynamic viewport (minimum 320px), and the place list follows beneath. Selecting a place opens its menu board as a fixed bottom sheet (up to 60% viewport height) instead of replacing the list; the map pans so the selected sign stays above the sheet.

Spacing runs on a 4px grid with 8px and 12px as the working gaps: 12px side padding on phones, 20px from 768px; 8px between plates; 12px–16px inside rows and boards. Touch targets are 40px (filter plates, zoom buttons) or 44px (header fields and the primary plate).

## Elevation & Depth

The world is flat: panels sit on the page with borders, not shadows. Depth appears only for elements that physically float above the map or the page (pins, the zoom plate, the empty-state card, the mobile sheet, the notice-board dialog), and there it is soft, ambient, ink-tinted shadow, never a hard offset.

### Shadow Vocabulary
- **Sign cast** (`box-shadow: 0 3px 6px rgb(26 26 23 / 0.35)`): under each map sign plate.
- **Control cast** (`box-shadow: 0 3px 8px rgb(26 26 23 / 0.25)`): the map zoom plate.
- **Card lift** (`box-shadow: 0 8px 24px rgb(26 26 23 / 0.2)`): the empty-state card over the map.
- **Sheet lift** (`box-shadow: 0 -10px 30px rgb(26 26 23 / 0.3)`): the mobile bottom sheet, cast upward.
- **Dialog lift** (`box-shadow: 0 24px 60px rgb(0 0 0 / 0.35)`): the notice-board dialog, over a 60% ink backdrop.

The auth board lies flat on its green wall and casts no shadow.

### Named Rules
**The Only-Floaters-Cast Rule.** Boards, strips, lists, and the desktop menu board have no shadow. A shadow means "this is lifted above the map"; on desktop the details panel drops its sheet shadow because it no longer floats.

## Shapes

Signboards are rectangles with barely softened corners: 3px for every plate, field, and button; 2px for the inner code square; 5px for outer frames (wordmark frame, menu board, empty-state card); 6px for the dialog; 10px for the mobile sheet's top corners only. The signature form is the **painted inset border**: an outer plate or frame with ~4–6px padding holding an inner rectangle outlined in a thinner or softer line (yellow 2px on the wordmark and notice-board header; ink at 40%, 1px inside the menu board). Borders are 2px ink by default; dashed marks "off" or "missing", dotted marks menu-board leaders. The only full round is the grab handle on the mobile sheet.

## Components

### Buttons
Painted plates, confident and flat.
- **Shape:** squared plate (3px), 2px ink border.
- **Primary (sign plate):** sign yellow with ink Bungee lettering, 44px tall in the header (40px in overlays), 16px side padding.
- **Hover / Focus:** fill fades to faded sign yellow (colour transition); focus is a 2px outline offset 2px, yellow on the green board, painted blue on paint white. Disabled drops to 50% opacity.
- **Quiet buttons** (close, back, clear, zoom): no border, ink icon or label, faded-yellow fill on hover and focus, with an inset blue focus outline; on the green board they hover to deep board green.
- **Icons:** Lucide line icons at a 2.5 stroke, 16–20px.
- **Submit plate (forms):** a full-width sign plate, 44px, ink Bungee at 0.875rem, blue focus outline offset 2px (it sits on paint white). Hover fades to faded sign yellow. While pending it is disabled, holds the faded sign-yellow fill, and reads "Memproses…".

### Chips (category filter plates)
- **Style:** 40px plate with a small code square on the left and a Figtree bold label.
- **Inactive:** paint-white fill, dashed 2px outline in the category hue (ink at 60% for Laundry and "Semua"); the code square is a solid category plate. Hover turns the dashed outline solid.
- **Active:** the whole plate fills solid with the category paint and paint-white (or ink, for yellow) lettering; the code square becomes a 20% paint-white tint. "Semua" active is a solid ink plate. Plates toggle independently, and "Semua" is active when no category is filtered.

### Cards / Containers
- **Menu board (place details):** a 5px paint-white frame with 2px ink border and 4px padding, holding a 3px inner panel ringed in 1px ink at 40%. Inside: the category badge, Bungee headline, Figtree description, a 2px ink rule, then leader rows (label, 2px dotted leader in ink at 25%, right-aligned value) and a dashed-divided address block. The category's custom fields follow the price row in the category's field order, rendered by data type: short values (numbers, yes/no led by a 16px check or cross icon, choices, ranges as "12–24", dates as "2 Maret 2015") as leader rows, then free text as dashed-divided blocks like the address, and a missing value as a missing slot reading "Data tidak tersedia".
- **Notice board (kecamatan info):** a native modal dialog (6px, 2px ink border) with a sticky green header whose yellow-inset plate carries the Bungee title and close button, a 6px chili stripe, then dashed-divided sections each headed by a green icon and a bold label.
- **Empty-state card:** 5px paint-white card with 2px ink border and card lift, centred over a 70% paint scrim on the map, with one message and one yellow action plate.

### Inputs / Fields
- **Search:** paint-white field, 2px ink border, 3px corners, 44px tall, leading search icon in soft ink, trailing quiet clear button when filled; the native cancel control is hidden.
- **Kecamatan select:** deep-board-green field with a 2px paint-at-35% border (60% on hover) and a yellow chevron; options render on paint white.
- **Focus:** fields swap their border to sign yellow instead of drawing an outline.
- **Form field (on paint white):** a Figtree bold label (0.875rem, ink) above a 44px paint-white field with a 2px ink border, 3px corners, and 12px side padding. Focus is a 2px painted-blue outline offset 2px.
- **Form field, invalid:** after a submit, the border turns chili and a chili message (0.875rem semibold) sits under the field; editing the field clears it. Without an error, an optional soft-ink hint sits in the same slot.
- **Form alert:** a form-level message plate with a 2px border and 3px corners (10px × 12px padding), no fill, and ink semibold text at 0.875rem. It carries errors only: a chili border and chili alert icon (an assertive alert). It sits at the top of the form, above the fields.

### Navigation
- **Header board:** full-width warung green with the wordmark plate (deep green frame, yellow inset border, yellow Bungee), a paint-at-85% tagline, and the controls; closed by a 6px chili stripe.
- **Category strip:** paint white with a 2px ink bottom rule; the count sentence is a polite live region with the number in Bungee.
- **Account plate (header):** a 44px plate on the green board with a 2px paint-at-35% outline, no fill, and a paint-white Figtree bold label (0.875rem) led by a 16px user icon. Hover raises the border to paint at 60% and fills deep board green. Focus is a 2px yellow outline offset 2px. It is never a yellow plate, because yellow is reserved for the action plate. Signed out it reads "Masuk" and links to the sign-in page, carrying the current map address. Signed in, the first name (paint-white bold, truncated at 10ch) sits beside the same plate reading "Keluar" with a log-out icon.
- **Place list:** rows divided by 1px dashed ink at 20%; each row is a code plate, name (Figtree semibold), truncated address (soft ink), and right-aligned tabular price; hover and focus fill with faded sign yellow at 60%.

### Map Sign (signature)
A place is a 36px category plate (2px border, Bungee code, sign cast) on a 3px × 10px ink post, anchored at the post's foot. Hover lifts the plate 2px. Selected inverts it: ink fill, yellow border and code, scaled to 115%, raised above other signs. Keyboard focus draws a blue outline around the sign.

**Planting motion:** when signs appear, each rises from its post (vertical scale from 0 at the base) over 500ms with `cubic-bezier(0.16, 1, 0.3, 1)`, staggered west to east in 40ms steps up to 360ms. Changing kecamatan flies the map to the new bounds over 0.9s; reduced motion jumps instead and skips the rise.

**Map canvas:** Esri Light Gray base and reference tiles (streets and labels, no POI icons) warmed with a 30% sepia filter over a deep-paint backing; the default Leaflet zoom control is replaced by a paint-white zoom plate (2px ink border, ink divider, control cast) at the top left; attribution sits on paint at 90% in 10px soft ink with blue links.

### Auth Wall (Masuk / Daftar)
The header board grown to fill the viewport: full-bleed warung green with paint-white text. It has no chili stripe, because it never meets a paint-white strip, so there is no seam to close. The wordmark plate and paint-at-85% tagline are the map header's own (one shared component; the tagline hides below 640px on both), and the wordmark links back to the map.
- **Board:** a menu-board-framed paint-white board (5px frame, 2px ink border, 4px padding, inner 3px panel ringed in 1px ink at 40%, 16px inner padding, 20px from 640px). It is flat and casts no shadow. The plate pair and lede sit 12px apart, and the form follows 20px below.
- **Plate pair:** a two-column row of equal 48px plates with an 8px gap, lettered in ink Bungee at 1.125rem. The current page's plate is a solid sign-yellow plate with a 2px ink border and is the page's `<h1>`. The other plate is a paint-white link with a dashed 2px ink-at-60% outline that turns solid ink on hover, with a blue focus outline offset 2px. Switching plates swaps only the form; the board and plates stay put.
- **Lede:** one soft-ink line at 0.875rem under the plates.
- **Form:** fields, alert, and submit plate stacked at a 16px gap (see Inputs / Fields and Buttons).
- **Return link:** "Kembali ke peta" under the board, a 40px quiet link on the green: paint-white semibold 0.875rem with a 16px back arrow, deep board green on hover and focus, and a 2px yellow focus outline. It and the wordmark return to the exact map address the visitor came from.

### Missing Slot
A drawn empty slot: inline dashed 2px border in ink at 30%, 3px corners, soft-ink label-size text such as "Jam buka tidak tersedia" or "Data tidak tersedia". A missing area image becomes a dashed 8:5 frame holding the slot.

### Motion
The mobile sheet slides up from off-screen over 300ms ease-out; the notice board fades in and rises 16px over 300ms ease-out; hovers transition colour at the default 150ms. The auth board settles into place on entry by translating up 12px over 500ms with the planting ease `cubic-bezier(0.16, 1, 0.3, 1)`, with no fade. All motion is gated on `prefers-reduced-motion: no-preference`.

## Do's and Don'ts

### Do:
- **Do** style only with Tailwind v4 utilities and the `@theme` tokens in `app/app.css`; never write raw CSS rules or inline styles.
- **Do** letter every sign-like element (wordmark, codes, sign headings, action plates, money and counts) in Bungee and everything read at length in Figtree.
- **Do** show state by the mark: solid plate for on, dashed outline for off, ink inversion with a yellow code for selected.
- **Do** keep each category in its one paint and code (W chili, J plum, K blue, L yellow, A green) across plates, pins, list, and badges.
- **Do** draw missing data as a dashed missing slot with explicit Bahasa Indonesia copy, never an empty space.
- **Do** close a green board with the 6px chili stripe, and frame a sign title with the painted inset border.
- **Do** use 2px ink borders and 3px corners for plates and fields; 40–44px touch targets.
- **Do** gate every animation on reduced-motion preference.

### Don't:
- **Don't** use gradients, glossy fills, or glass effects; panels are flat enamel.
- **Don't** use white floating cards, pill-shaped chips, or blue droplet pins; plates are rectangles and pins are signs on posts.
- **Don't** letter paint-white text on sign yellow or use yellow as an outline on paint white.
- **Don't** use hard offset shadows; shadows are soft, ink-tinted, and reserved for elements floating over the map.
- **Don't** set body copy, labels, or form values in Bungee.
- **Don't** add map markers or tile styles that compete with the painted signs; keep the canvas quiet and POI-free.
