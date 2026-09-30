# Stallfunk design system

Warm, calm, flat, like a notice board in the stable aisle: text on paper, a few white cards
for the things that are objects (a horse, a request, a list), one green button for the one
thing to do. Built with NativeWind v4 (Tailwind 3) in the shadcn / react-native-reusables
style: components are plain source files in `mobile/components/ui/`, owned by this repo.
Tokens live in one place, `mobile/lib/tokens.ts`, and are mapped to Tailwind by
`mobile/tailwind.config.ts`.

Code and identifiers are English; all UI text is German.

## Rules

1. **Every screen opens with a `PageHeader`**: title in Fraunces, optionally a key figure and one
   sentence, set directly on the page background. No colored block, no box. The one exception is
   `LivePanel`, the dark readout of a running session (the gait colors are made for it).
2. **Cards are for objects, not for layout.** A horse, a request, a reminder list, a settings
   group: `Card`. Forms, buttons, inputs, explanations and empty states sit directly on the page.
   Never a card inside a card; a tile inside a card only for a small grid (blankets, health).
3. **Headings only where a screen really has groups** (`Section` / `SectionTitle`, Fraunces 20 px).
   A single list or form needs none. Field names in forms use `Text variant="label"`.
4. **Emphasis comes from type and the one primary button**, not from painting a block. Filled
   surfaces are reserved for meaning: `danger-soft` for an error, `accent-soft` for a warning,
   `primary-soft` for a selected item.
5. **No emojis, no gradients, no shadows.** Depth comes from a 1 px `border-border`.
6. **Touch targets are at least 44 px** (`min-h-touch`, `h-11`). Small visuals get `hitSlop`.
7. Page padding is 24 px, section gap 24 px, inside a section 12 px. `Screen` and `Section` do this;
   do not add your own.
8. Colors come from tokens only. The default Tailwind palette (zinc, gray, ...) is removed on purpose.
9. Never set `fontWeight`. React Native needs one font family per weight; use `Text` variants or `font-sans-semibold` etc.
10. Icons are Lucide, stroke 2, through `Icon`.
11. Use `cn()` (`@/lib/cn`) to merge classes. Do not use `StyleSheet` for things that have a token.
12. Texts stay short (JAN-80): titles of one to three words, one sentence below, buttons of one or two words.

## Tokens

### Colors (Tailwind names: `bg-*`, `text-*`, `border-*`)

| Token | Value | Use |
| --- | --- | --- |
| `background` | `#f6f4ee` | Screen background |
| `card` | `#ffffff` | Cards, tab bar, inputs |
| `border` | `#e7e2d9` | 1 px borders (default for `border`) |
| `divider` | `#f0ebe1` | Hairlines inside cards, neutral badge |
| `foreground` | `#1c1917` | Primary text |
| `muted` | `#6b6560` | Secondary text |
| `primary` / `primary-soft` / `primary-deep` / `primary-deeper` | `#2d5a3d` / `#e3efe6` / `#24503a` / `#1c3a27` | Actions, active state, wordmark, `LivePanel` |
| `accent` / `accent-soft` / `accent-text` | `#c2410c` / `#fdf1e0` / `#9a4d0b` | Warnings, highlights |
| `info` / `info-soft` | `#1e4f8a` / `#e6eefb` | Informational |
| `danger` / `danger-soft` | `#9f1d1d` / `#fde8e8` | Destructive, errors |
| `gait-walk` / `gait-trot` / `gait-canter` | `#86efac` / `#fbbf24` / `#ffffff` | Gaits in the `LivePanel` |
| `gait-canter-light` | `#c2410c` | Canter on light backgrounds |

Values needed in JS (`style`, navigation options): `import { colors } from "@/lib/theme"`.

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
`resolveColorKey(key)`, `isColorKey(v)`, `COLOR_KEYS`, `COLOR_PAIRS`. `@/lib/initials`: `initialOf(name)`.

### Typography

Geist (UI, 400 to 700) and Fraunces (titles, headings, key figures, the wordmark), loaded in `app/_layout.tsx` with `expo-font`;
the splash screen stays until they are ready. Class names for fonts: `font-sans`, `font-sans-medium`,
`font-sans-semibold`, `font-sans-bold`, `font-display`, `font-display-medium`, `font-display-bold`.

| Size class | px | Used by |
| --- | --- | --- |
| `text-caption` | 12 / 16 | `caption`, badges |
| `text-secondary` | 13 / 18 | `secondary`, `label` |
| `text-body-sm` | 14 / 20 | `bodySm`, buttons (sm) |
| `text-body` | 15 / 22 | `body`, `bodyStrong` |
| `text-heading` | 20 / 26 | `heading` (Fraunces): `SectionTitle`, wordmark |
| `text-title` / `text-title-lg` | 26 / 28 | `title`, `titleLg` (Fraunces): sheets, prominent cards |
| `text-title-xl` | 32 / 38 | `display` (Fraunces): `PageHeader` title, login code |
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
  `edges` (default `["top"]`), `contentClassName`, `refreshControl`, `className`. Children are spaced 24 px apart.
  Tab screens use the default; sub pages (no tab bar) use `back`. Screens without a session
  (sign-in, join) use `contentClassName="flex-grow pt-8"` and push their footer down with `mt-auto`.
- **`PageHeader`**: the head of every screen, on the page background. Props: `eyebrow`, `title` (Fraunces 32),
  `action` (element right of the title, e.g. an icon `Button` or an `Avatar`), `value` + `valueSize` (`sm|md|lg`) + `unit`
  (the key figure), `description`, `children` (badges, a progress bar, a small button). Exactly one per screen.
- **`LivePanel`**: dark green readout for a running session only (`eyebrow`, `title`, `value` 72 px, `description`, `children`
  in white). The only filled block in the app.
- **`Section`**: `<Section title="Noch offen" action={...} description="...">…</Section>`: `SectionTitle` (Fraunces 20) plus its
  content, 12 px apart. **`SectionTitle`** alone when the content is not a simple child list.
- **`BackButton`**: round 44 px, goes back or to `/`. Normally rendered by `Screen back`.
- **`Card`**: `shape` (`card` 20 | `tile` 16), `padded` (default true, 20 px). **`PressableCard`**: same, tappable.
  **`Divider`**: hairline inside cards; with `label="oder"` a labeled rule between two groups on the page.
- **`Brand`** (`@/components/brand`): logo and wordmark, as in the login mail. Sign-in and the magic-link screen only.

### Text and icons

- **`Text`**: `variant` = `display | heading | title | titleLg | heroNumberSm | heroNumber | heroNumberLg | body | bodySm | bodyStrong | secondary | caption | label`,
  `tone` = `default | muted | primary | accent | info | danger | inverse`. Use `className` for the rest.
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

### Display and overlays

- **`Badge`**: static status label. `label`, `variant` (`neutral|primary|accent|info|danger`).
- **`Avatar`**: `name`, `colorKey`, `size` (`sm` 32, `md` 44, `lg` 64, `xl` 96).
- **`Sheet`**: bottom sheet. `open`, `onOpenChange`, `title`, `description`, children.

## Navigation

Bottom tab bar (`app/(tabs)/_layout.tsx`): Start, Decken, Anfragen, Training, Pferde with Lucide icons
(`House`, `Shirt`, `HandHelping`, `Activity`, `PawPrint`). The active tab is a green icon inside a `primary-soft` pill;
the bar is white with a 1 px top border. Its dimensions are in `tabBar` in `@/lib/theme`.
Sub pages sit outside `(tabs)` (no tab bar) and use `<Screen back>`.

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
