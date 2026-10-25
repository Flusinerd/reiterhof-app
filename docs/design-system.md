# Stallfunk design system ("Abendlicht")

Calm and warm, built around the day of a horse owner: the app says what is up for your horse
now, not where a feature lives. It follows the phone's light or dark setting. Light is warm paper
by day; dark ("Abendlicht") is the stable at night, with brass for the one thing to do. Built with
NativeWind v4 (Tailwind 3) in the shadcn / react-native-reusables style: components are plain
source files in `mobile/components/ui/`, owned by this repo. Tokens live in one place,
`mobile/lib/tokens.ts`; `mobile/global.css` carries them as CSS variables per scheme and
`mobile/tailwind.config.ts` maps the Tailwind colors to those variables.

Code and identifiers are English; all UI text is German. The concept and the decisions behind it
(concept "Mischung", owners first, Abendlicht, optional horse photos) were made with the product
owner in October 2026; the rebuild goes step by step: foundation and "Heute", then the horse page
with photo and diary, then the flows (after the ride, evening round; built), then the remaining screens.

## Rules

1. **One next thing, then a timeline.** Screens that answer "what now" lead with one raised block
   (`bg-raised`) holding the next task and its single primary button, followed by rows with
   hairlines (`border-divider`), not a stack of cards.
2. **One primary action per screen** (brass in the dark, deep green in the light). Everything else is
   `outline`, `ghost` or a tappable row.
3. **Leave out what is empty.** No "Keine offenen Anfragen" blocks on overview screens; an empty
   state belongs on the screen that lists the thing.
4. **Titles have a voice:** `PageHeader` titles, section headings and horse names are Fraunces
   italic (`display`, `heading`, `title`). Everything that is operated or scanned is Geist.
5. **Times and numbers in brass** (`tone="brass"`), states in `success` (done), `accent`
   (warning), `danger` (overdue, errors). Green is no longer the general highlight.
6. **Every screen opens with a `PageHeader`** with a small upper-case `eyebrow` (date, horse, stable).
7. **Cards are for objects** (a horse, a request, a settings group), never for layout. Never a card
   inside a card.
8. **No emojis, no gradients, no shadows.** Depth comes from the `raised` surface and 1 px borders. The one exception is the
    horse photo, which fades into the page background (`HorseHero`).
9. **Touch targets are at least 44 px** (`min-h-touch`, `h-11`). Small visuals get `hitSlop`.
10. Page padding is 24 px, section gap 24 px, inside a section 12 px. `Screen` and `Section` do this.
11. **Colors come from tokens only**, as class names. JS values (spinner tints, the tab bar, native
    props) come from `useColors()` (`@/lib/theme`), which follows the scheme; `colors` from
    `@/lib/theme` is the current palette read at render time. The default Tailwind palette is removed.
12. Never set `fontWeight`; use `Text` variants or `font-sans-semibold` etc.
13. Icons are Lucide, stroke 2, through `Icon`; the custom `Horse` (`@/components/icons/horse`) is drawn
    in the Lucide style.
14. Use `cn()` (`@/lib/cn`) to merge classes. Texts stay short (JAN-80).
15. **Flows that belong to the night are always dark** (the evening blanket round), however the phone is set:
    wrap them in `ThemeScope` (see "Always-dark flows"). Everything else follows the phone.

## Tokens

### Colors (Tailwind names: `bg-*`, `text-*`, `border-*`)

`palettes.light` / `palettes.dark` in `lib/tokens.ts`; `global.css` has the same values as RGB
channels (`--color-<name>`), so opacity modifiers work (`bg-primary/20`). `lib/tokens.test.ts`
checks that both palettes have the same tokens and that `global.css` matches.

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `background` | `#f4f0e6` | `#0e1210` | Screen background, tab bar |
| `card` | `#fffdf8` | `#141a16` | Cards, inputs, sheets |
| `raised` | `#fffdf8` | `#19211c` | The next task, running flows |
| `border` / `divider` | `#e6dfd1` / `#eee8db` | `#2a342d` / `#1f2722` | Borders / hairlines between rows |
| `foreground` / `muted` | `#1a1814` / `#6d675c` | `#ede7da` / `#98a294` | Text / secondary text |
| `primary` (+ `foreground`, `soft`, `deep`, `deeper`) | `#1f3527` | `#c9a45c` | The one action; text on it is `primary-foreground` |
| `brass` / `brass-soft` | `#8a6a2c` | `#c9a45c` | Times, numbers, active tab |
| `success` / `success-soft` | `#2d6a45` | `#8cc49e` | Done, in good shape, switches |
| `accent` (+ `soft`, `text`) | `#c2410c` | `#e59457` | Warnings |
| `info` / `info-soft` | `#1e4f8a` | `#8fb3e6` | Informational |
| `danger` / `danger-soft` | `#9f1d1d` | `#e48585` | Overdue, destructive, errors |
| `panel` | `#16241b` | `#19241d` | `LivePanel` (dark in both schemes) |
| `gait-walk` / `-trot` / `-canter` / `-canter-light` | | | Gaits (same in both schemes) |

### Horse and person colors

Backend stores a `color_key` per horse/user. Keys: `green`, `amber`, `blue`, `rose`, `violet`, `teal`, `neutral`.

| Key | Background | Foreground | Horses |
| --- | --- | --- | --- |
| `green` | `#a3c9ad` | `#1c3a27` | Luna |
| `amber` | `#f0c380` | `#4a2a05` | Fanta, Nala |
| `blue` | `#b9c7ea` | `#1e3560` | Balu |
| `rose` | `#f2b8c6` | `#5a1a2c` | Cookie |
| `violet` | `#cbb8e8` | `#3b1f63` | Merlin |
| `teal` | `#9fd6cf` | `#0f3f3a` | Pepe |
| `neutral` | `#e7e2d9` | `#44403c` | People without a key, unknown keys |

Helpers in `@/lib/color-keys`: `colorsForKey(key)` returns `{ bg, fg }` (unknown, empty or missing keys give `neutral`),
`resolveColorKey(key)`, `isColorKey(v)`, `COLOR_KEYS`, `COLOR_PAIRS`. In the dark scheme the pastels would glow, so
`colorsForKey(key, "dark")` returns `DARK_COLOR_PAIRS` (a muted dark tint with a light initial, e.g. green `#2c4434` / `#b9dcc3`);
`Avatar`, the timeline dots of "Heute" and the color picker pass `useColorScheme()`. Light is unchanged. `@/lib/initials`: `initialOf(name)`.

### Typography

Geist (UI, 400 to 700) and Fraunces (upright 500 to 700 for key figures and the wordmark, light and medium
italic for titles), loaded in `app/_layout.tsx` with `expo-font` (`lib/fonts.ts`);
the splash screen stays until they are ready. Class names for fonts: `font-sans`, `font-sans-medium`,
`font-sans-semibold`, `font-sans-bold`, `font-display`, `font-display-medium`, `font-display-bold`,
`font-display-italic`, `font-display-italic-medium`.

| Size class | px | Used by |
| --- | --- | --- |
| `text-caption` | 12 / 16 | `caption`, badges |
| `text-secondary` | 13 / 18 | `secondary`, `label` |
| `text-body-sm` | 14 / 20 | `bodySm`, buttons (sm) |
| `text-body` | 15 / 22 | `body`, `bodyStrong` |
| `text-heading` | 20 / 26 | `heading` (Fraunces medium italic): `SectionTitle` |
| `text-title` / `text-title-lg` | 26 / 28 | `title`, `titleLg` (Fraunces light italic): sheets, the next task |
| `text-[11px]` upper case | 11 / 16 | `eyebrow` (Geist 600, letter spacing 1.4): the line above titles |
| `text-title-xl` | 34 / 38 | `display` (Fraunces light italic): `PageHeader` title |
| `text-hero-sm` / `text-hero` / `text-hero-lg` | 44 / 56 / 72 | `heroNumberSm`, `heroNumber`, `heroNumberLg` (Fraunces) |

### Shape and layout

| Class | Value |
| --- | --- |
| `rounded-card` | 20 (cards, live panel, sheet) |
| `rounded-tile` | 16 (tiles) |
| `rounded-button-sm/md` / `rounded-button-lg` | 12 / 14 |
| `rounded-pill` | 9999 |
| `px-page` / `p-page` | 24 |
| `min-h-touch`, `min-w-touch` | 44 |

## Components

All exported from `@/components/ui` (barrel) or individually from `@/components/ui/<name>`.

### Layout

- **`Screen`**: root of every screen. Props: `scroll` (default true), `back` (sub page: shows `BackButton`, pads bottom safe area),
  `menu` (default true), `edges` (default `["top"]`), `contentClassName`, `refreshControl`, `className`. Children are spaced 24 px apart.
  Tab screens use the default; sub pages (no tab bar) use `back`. A top bar holds the `BackButton` (left) and the app menu
  (right, `components/app-menu.tsx`: "Bin da" / "Ich gehe", Erinnerungen, Einstellungen). The root layout provides the menu
  through `ScreenMenuContext` while someone with a stable is signed in; `menu={false}` hides it (running ride tracking). Screens without a session
  (sign-in, join) use `contentClassName="flex-grow pt-8"` and push their footer down with `mt-auto`.
- **`PageHeader`**: the head of every screen, on the page background. Props: `eyebrow`, `title` (Fraunces 32),
  `action` (element right of the title, e.g. an icon `Button` or an `Avatar`), `value` + `valueSize` (`sm|md|lg`) + `unit`
  (the key figure), `description`, `children` (badges, a progress bar, a small button). Exactly one per screen.
- **`LivePanel`**: dark readout (`bg-panel`) for a running session only (`eyebrow`, `title`, `value` 72 px, `description`, `children`
  in white). The only filled block in the app.
- **`Section`**: `<Section title="Noch offen" action={...} description="...">…</Section>`: `SectionTitle` (Fraunces 20) plus its
  content, 12 px apart. **`SectionTitle`** alone when the content is not a simple child list.
- **`BackButton`**: round 44 px, goes back or to `/`. Normally rendered by `Screen back`.
- **`Card`**: `shape` (`card` 20 | `tile` 16), `padded` (default true, 20 px). **`PressableCard`**: same, tappable.
  **`Divider`**: hairline inside cards; with `label="oder"` a labeled rule between two groups on the page.
- **`LinkRow`**: navigation row with `icon`, `label`, optional `description`, `onPress` and a chevron (44 px minimum).
  Stack several in one `Card padded={false}` with `Divider`s between them.
- **`ListRow`** + **`RowList`** (`@/components/ui/list-row`, import it directly, not from the barrel): a row on the page background
  without a card: `leading` (avatar, icon, number), `eyebrow` (+ `eyebrowTone`), `title`, `detail`, `trailing` (outside the tap area,
  e.g. a quiet `Button` or a value in brass), `children` below the text, `onPress` (tap area with chevron). `RowList` puts a hairline
  between rows (empty children are dropped, so do not pass components that render `null`). Used by training, horse sub pages,
  observations, reha, blanket plan and settings; prefer it over `LinkRow` in `Card`s.
- **`Brand`** (`@/components/brand`): logo and wordmark, as in the login mail. Sign-in and the magic-link screen only.

### Text and icons

- **`Text`**: `variant` = `display | heading | title | titleLg | heroNumberSm | heroNumber | heroNumberLg | body | bodySm | bodyStrong | secondary | caption | label | eyebrow`,
  `tone` = `default | muted | primary | brass | success | accent | info | danger | inverse`. Use `className` for the rest.
- **`Icon`**: `<Icon as={Check} size={20} className="text-primary" />`. Default size 24, stroke 2. Color via `text-*` class.

### Actions and inputs

- **`Button`**: `label`, `variant` (`primary|secondary|outline|ghost|danger`), `size` (`sm` 44, `md` 48, `lg` 56, `icon` 44 x 44), `icon`, `loading`, `disabled`, `fullWidth`, plus all `Pressable` props. Icon-only buttons need `accessibilityLabel`.
- **`Input`**: single-line text field (48 px, white surface, 1 px border), all `TextInput` props. Always set `accessibilityLabel`.
  Sits directly on the page, never in a card. One-time codes are set large and centered in Fraunces
  (`className="h-20 text-center font-display text-title-xl"`, `letterSpacing` 6).
- **`Switch`**: `value`, `onValueChange`, optional `label` / `description` (whole row tappable), `disabled`.
- **`ToggleGroup`** + **`ToggleGroupItem`**: `type="single"` (`value: string`) or `type="multiple"` (`value: string[]`), items have `value`, `label`, `icon`.
- **`Tabs`**, **`TabsList`**, **`TabsTrigger`**, **`TabsContent`**: in-page segmented tabs, controlled with `value` / `onValueChange`.
- **`Pill`**: tappable chip (`label`, `selected`, `icon`) for filters. 36 px visual, 44 px touch target.
- **`Stepper`**: progress of a multi-step flow (`steps`, `current` zero-based, `label`): equal bars and "Schritt 2 von 5 · Aktivitäten".
  Sits inside the `PageHeader` of the wizard screens.
- **`NumberStepper`**: whole number with "Weniger" / "Mehr" icon buttons (`value`, `min`, `max`, `onChange`, `accessibilityLabel`, `format`).
  **`RangeStepper`** (same file): `label`, `min`, `max`, `lowerBound`, `upperBound`, `onChange(min, max)`; two steppers joined by "bis",
  the other end is dragged along when they would cross.
- **`DateField`**: date field with the native picker (Android dialog, iOS wheel, web `<input type="date">`). `value` is `"YYYY-MM-DD"` or `""`,
  shown as "17.05.2026"; `accessibilityLabel`, `placeholder`, `clearable`, `minimumDate`. Sibling of `TimeField`, same look as `Input`.

### Display and overlays

- **`ThemeScope`**: `<ThemeScope scheme="dark">…</ThemeScope>` (default `dark`) pins a subtree to one scheme, see "Always-dark flows".

- **`Badge`**: static status label. `label`, `variant` (`neutral|primary|accent|info|danger`).
- **`Avatar`**: `name`, `colorKey`, optional `photoUrl` (`Horse.photo_url`, shown instead of the initial, falls back to it if the image does not load),
  `size` (`sm` 32, `md` 44, `lg` 64, `xl` 96).
- **`Segmented`**: `items [{value, label}]`, `value`, `onValueChange`. Underline switch between the views of one screen (hairline, the selected
  label with a brass underline), quieter than `Tabs`; the screen renders the content itself. Used on the horse page.
- **`AuthImage`** (`@/components/auth-image`): `url` of a stored file (`/api/v1/files/...`) plus `Image` props; adds the session token on
  every platform (header on native, fetched blob URL on the web). Use it instead of `<Image source={fileSource(url)}>`.
- **`Sheet`**: bottom sheet. `open`, `onOpenChange`, `title`, `description`, children.

### The horse page

`app/horses/[id]/index.tsx` is the model of a "page of an object": a full-bleed `HorseHero` (photo, or the horse's color with its big
italic initial as ornament; eyebrow, name in `display` 44 px), then `Section "Heute für <Name>"` with rows (`HorseTodayRows`: eyebrow in
a tone, title, detail, at most one `Button sm` on the right; only the training start is `primary`), `Section "Diese Woche"` with
`HorseWeekStrip` (done `success`, planned dashed, show day `brass`), then `Segmented` and its content. The diary (`HorseDiary`) is a
timeline: a hairline with one dot per entry (`success` session, `accent` open observation, `danger` urgent, `border` done), the line
above in `eyebrow` ("Gestern · Mia"), notes as italic quotes (`font-display-italic`), photos as 72 px thumbnails.

### The stable pages

Everything about the stable follows the same pattern: no cards, rows on the page with a hairline each (`border-b border-divider`,
the last one without). A person is a row (`Avatar`, name, a short line, something in brass on the right: "seit 18:40"). A request is a
message (`RequestRow`: avatar, "Anna · morgen 07:00" with the time in brass, the text, a quiet line, an outlined answer button),
grouped by day under a `SectionTitle`. A blanket horse is a row with the blanket in italic and the state buttons in one compact
row (`flex-1`, `size="sm"`, 8 px padding, so three labels fit at 390 px); finished rows fade (`opacity-60`). A reminder is the same line
as on "Heute". The weather is one sentence and three figures in brass (`WeatherCard`); a warning is a row with a stripe in the
tone of its level. Form labels are `eyebrow`; the choices of a type are `Pill`s with an icon; the page has one primary button
("Senden", "Neu", "Runde starten", "Bin da"). Selected text on a `bg-primary` surface is `text-primary-foreground`, never `text-white`
(brass in the dark scheme needs dark text).

## Feedback

How a screen answers a tap. No toast component is needed for this; if one is added later it needs its own entry here.

1. A "done" action turns into its success state in place: check icon + `tone="success"` text ("Heute erledigt",
   "Erledigt von Mia") + ghost "Rückgängig" where the server can undo; the triggering button disappears
   (model: `components/reha-today-head.tsx`).
2. Rows do not move or change identity under the finger while the user is on the screen (freeze the order per
   visit; show the acknowledged item until blur).
3. Switches and pills that save at once are optimistic with rollback (model: `app/settings/index.tsx`).
4. A button's spinner lasts until the screen shows the new data (await the invalidation or write the response into
   the cache).
5. A flow that leaves the screen lands where the result is visible (right params, `router.dismissTo`) or shows a
   success state with one "Fertig" first (models: finish "Im Tagebuch", report "Gemeldet"). A flow that removes the
   screen's own subject (delete a horse) leaves every screen of it and names the result on the landing screen
   (`lib/flash.ts`, `FlashNotice`: "Luna wurde gelöscht."), and its effect removes the subject's queries instead of
   refetching them (`horseDeleted`, models: `components/horse-delete-sheet.tsx`).
6. Errors appear next to the action that failed; a failed query never renders as an empty or "alles erledigt" state.

### Checklist for every new mutation

1. Which key families does the server change (including derived ones: reminders, diary, training horses, Heute
   inputs)? Add or extend an `Effect` in `lib/query-effects.ts` with a test.
2. Is the invalidation awaited (or the response written with `setQueryData`) so the pending state lasts until the
   screen is fresh?
3. Can a repeat (double tap, retry, stale screen) write wrong data? Then make the endpoint idempotent (no-op on
   repeat, precondition, or `client_id`) and replace the button with a success state.
4. Does the trigger show pending for exactly that item, and is nothing else on screen able to repeat it?
5. Does the change matter to other members? Publish `<domain>.changed` with ids only and add it to the event map.
6. Where does the user end up, and does that screen show the result?
7. Where is the error shown, and is it visible from where the user tapped?
8. Toggle or pill? Optimistic with rollback.
9. Docs (domain doc, design system for new patterns) and the CHANGELOG updated.

## Navigation

Bottom tab bar (`app/(tabs)/_layout.tsx`), four tabs:

| Tab | Route | Content |
| --- | --- | --- |
| Heute | `(tabs)/index` | Next task and timeline of today and tomorrow for the user's horses (`lib/today.ts`, `lib/use-today.ts`, `components/today-parts.tsx`) |
| *Name of the horse* / Pferde | `(tabs)/horse` | The user's only horse (owner or rider) directly, named after it; with several horses the horse list |
| Stall | `(tabs)/stable` | "Bin da", who is here (faces and one sentence), tonight's blankets as a board of horse avatars with a status dot, requests for help as messages, weather warnings (`lib/stable-tab.ts`, `components/stable-parts.tsx`, `components/request-row.tsx`) |
| Ich | `(tabs)/me` | Links (reminders, training, exercises, all horses), notifications, account (the settings screen) |

The active tab is brass with a short line above the icon; the bar has the page background and a hairline.
Blankets (`/blankets`), requests (`/requests`), training (`/training`) and the horse list (`/horses`) are
sub pages at the same URLs as before, so push `data.screen` links keep working. Tab screens have no app
menu (`menu={false}`); sub pages keep it (back button left, presence, reminders, settings right).

"Heute" merges three sources: the week plan of each own horse (training of today, shows of today and
tomorrow), tonight's blanket of each own horse at the stable's reminder time, and the reminder center
(health, medication, reha, helpers, weather; `training_plan` and `last_person` are left out because the
plan and blanket entries cover them). The next task is the user's own training of today, else the earliest
open timed task from now on, else something overdue.

A failed source is not an empty one (`todayLoad` in `lib/today.ts`): when a query failed and has nothing cached,
"Heute" shows "Heute konnte nicht geladen werden." with the reason and "Erneut versuchen" instead of "Für heute ist
alles erledigt." or the no-horse hint; when only a refresh failed, the cached view stays and a quiet "Nicht aktuell."
with the same retry sits above it. The Stall tab does the same for the blanket board (`LoadFailed` and `NotCurrent` in
`components/load-notice.tsx`).

## Always-dark flows

The token class names read the CSS variables of `global.css`, which follow the phone. `ThemeScope`
(`components/ui/theme-scope.tsx`) overrides them for its subtree: it builds the variables from `palettes[scheme]`
(`paletteVariables` in `lib/tokens.ts`, the same `--color-<token>: r g b` as `global.css`) and passes them to a wrapping
`View` with NativeWind's `vars()`, so every `bg-*`, `text-*` and `border-*` inside is dark. A context makes `useColors()`
return the same palette, so spinner tints and native props agree; use `useColors()`, not the static `colors`, in
components that can appear inside a scope. Two things the scope cannot do and the screen does itself: the status bar
(`<StatusBar style="light" />`) and the stack's `contentStyle` (`<Stack.Screen options={{ contentStyle: { backgroundColor: palettes.dark.background } }} />`).

On the web a `Modal` (our `Sheet`) is rendered outside the wrapper, so while a scope is mounted the variables are also
set on `<html>` and restored when it unmounts; a sheet opened inside the round is therefore dark as well.

The only always-dark screen so far is `/blankets/round` ([blankets.md](domains/blankets.md#evening-round-abendrunde)):
it happens in the stable at night, with the one brass button. "Wie war's?" after the ride (`/training/session/finish`)
follows the phone like every other screen.

## Adding a screen

```tsx
import { Card, PageHeader, Screen, Section, Text } from "@/components/ui";

export default function Example() {
  return (
    <Screen back>
      <PageHeader eyebrow="Luna" title="Beispiel" value="3" valueSize="sm" unit="offen" description="Ein Satz dazu." />
      <Section title="Liste">
        <Card>
          <Text variant="body">Ein Objekt</Text>
        </Card>
      </Section>
      <Text variant="body" tone="muted">
        Leerzustände und Erklärungen stehen ohne Karte auf der Seite.
      </Text>
    </Screen>
  );
}
```

## The pages without a session

Sign-in, "Postfach prüfen", "Stall beitreten" and the magic-link screen share one layout: `Brand`
(logo and wordmark) or the back button at the top, the `PageHeader`, the form directly on the page
(large input, one primary button), a `Divider label="oder"` before the social buttons, legal links at
the bottom. The https fallback page of the login mail (`GET /auth/verify`, `backend/internal/auth/handlers.go`)
uses the same tokens with system fonts and sends its own Content-Security-Policy.

All components are shown in the hidden dev route `/dev/kit` (`mobile/app/dev/kit.tsx`).
