# Reiterhof design system

Warm, calm, flat. Built with NativeWind v4 (Tailwind 3) in the shadcn /
react-native-reusables style: components are plain source files in
`mobile/components/ui/`, owned by this repo. Tokens live in one place,
`mobile/lib/tokens.ts`, and are mapped to Tailwind by `mobile/tailwind.config.ts`.

Code and identifiers are English; all UI text is German.

## Rules

1. Exactly **one `Hero` per screen**. Everything else is `SectionLabel` + `Card`.
2. **No emojis, no gradients, no shadows.** Depth comes from a 1 px `border-border`.
3. **Touch targets are at least 44 px** (`min-h-touch`, `h-11`). Small visuals get `hitSlop`.
4. Page padding is 24 px, section gap 24 px. `Screen` does both; do not add your own.
5. Colors come from tokens only. The default Tailwind palette (zinc, gray, ...) is removed on purpose.
6. Never set `fontWeight`. React Native needs one font family per weight; use `Text` variants or `font-sans-semibold` etc.
7. Icons are Lucide, stroke 2, through `Icon`.
8. Use `cn()` (`@/lib/cn`) to merge classes. Do not use `StyleSheet` for things that have a token.

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
| `primary` / `primary-soft` / `primary-deep` / `primary-deeper` | `#2d5a3d` / `#e3efe6` / `#24503a` / `#1c3a27` | Actions, active state, dark hero |
| `accent` / `accent-soft` / `accent-text` | `#c2410c` / `#fdf1e0` / `#9a4d0b` | Warnings, highlights |
| `info` / `info-soft` | `#1e4f8a` / `#e6eefb` | Informational |
| `danger` / `danger-soft` | `#9f1d1d` / `#fde8e8` | Destructive, errors |
| `gait-walk` / `gait-trot` / `gait-canter` | `#86efac` / `#fbbf24` / `#ffffff` | Gaits on dark backgrounds |
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

Geist (UI, 400 to 700) and Fraunces (titles, hero numbers), loaded in `app/_layout.tsx` with `expo-font`;
the splash screen stays until they are ready. Class names for fonts: `font-sans`, `font-sans-medium`,
`font-sans-semibold`, `font-sans-bold`, `font-display`, `font-display-medium`, `font-display-bold`.

| Size class | px | Used by |
| --- | --- | --- |
| `text-caption` | 12 / 16 | `caption`, badges |
| `text-secondary` | 13 / 18 | `secondary`, `label` |
| `text-body-sm` | 14 / 20 | `bodySm`, buttons (sm) |
| `text-body` | 15 / 22 | `body`, `bodyStrong` |
| `text-title` / `text-title-lg` | 26 / 28 | `title`, `titleLg` (Fraunces) |
| `text-hero-sm` / `text-hero` / `text-hero-lg` | 44 / 56 / 72 | `heroNumberSm`, `heroNumber`, `heroNumberLg` (Fraunces) |

### Shape and layout

| Class | Value |
| --- | --- |
| `rounded-card` | 20 (cards, hero, sheet) |
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
  Tab screens use the default; sub pages (no tab bar) use `back`.
- **`Hero`**: the one focal block. Props: `tone` (`forest` default, `deep`, `soft`, `warm`, `plain`), `eyebrow`, `title`, `value` + `valueSize` (`sm|md|lg`) + `unit`, `description`, `children`. Children inherit white text on dark tones.
- **`SectionLabel`**: `<SectionLabel action={...}>Heute</SectionLabel>`. 13 px / 600 muted.
- **`BackButton`**: round 44 px, goes back or to `/`. Normally rendered by `Screen back`.
- **`Card`**: `shape` (`card` 20 | `tile` 16), `padded` (default true, 20 px). **`PressableCard`**: same, tappable. **`Divider`**: hairline inside cards.

### Text and icons

- **`Text`**: `variant` = `title | titleLg | heroNumberSm | heroNumber | heroNumberLg | body | bodySm | bodyStrong | secondary | caption | label`,
  `tone` = `default | muted | primary | accent | info | danger | inverse`. Use `className` for the rest.
- **`Icon`**: `<Icon as={Check} size={20} className="text-primary" />`. Default size 24, stroke 2. Color via `text-*` class.

### Actions and inputs

- **`Button`**: `label`, `variant` (`primary|secondary|outline|ghost|danger`), `size` (`sm` 44, `md` 48, `lg` 56, `icon` 44 x 44), `icon`, `loading`, `disabled`, `fullWidth`, plus all `Pressable` props. Icon-only buttons need `accessibilityLabel`.
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
import { Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";

export default function Example() {
  return (
    <Screen>
      <Hero eyebrow="Heute" title="Beispiel" description="Ein Hero pro Screen." />
      <SectionLabel>Liste</SectionLabel>
      <Card>
        <Text variant="body">Inhalt</Text>
      </Card>
    </Screen>
  );
}
```

All components are shown in the hidden dev route `/dev/kit` (`mobile/app/dev/kit.tsx`).
