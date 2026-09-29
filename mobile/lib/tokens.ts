// Design tokens as plain data. This file has no React Native imports so that
// tailwind.config.ts, the app and the unit tests can all share it.
// Documentation: docs/design-system.md at the repo root.

/** Semantic color tokens. Keys map 1:1 to Tailwind color names. */
export const colors = {
  background: "#f6f4ee",
  card: "#ffffff",
  border: "#e7e2d9",
  divider: "#f0ebe1",
  foreground: "#1c1917",
  muted: "#6b6560",
  primary: {
    DEFAULT: "#2d5a3d",
    soft: "#e3efe6",
    deep: "#24503a",
    deeper: "#1c3a27",
  },
  accent: {
    DEFAULT: "#c2410c",
    soft: "#fdf1e0",
    text: "#9a4d0b",
  },
  info: {
    DEFAULT: "#1e4f8a",
    soft: "#e6eefb",
  },
  danger: {
    DEFAULT: "#9f1d1d",
    soft: "#fde8e8",
  },
  /** Gait colors. `canter` is for dark backgrounds, `canter-light` for light ones. */
  gait: {
    walk: "#86efac",
    trot: "#fbbf24",
    canter: "#ffffff",
    "canter-light": "#c2410c",
  },
} as const;

/** Radii in px. Cards 20, tiles 16, buttons 12-14, pills fully round. */
export const radii = {
  card: 20,
  tile: 16,
  "button-sm": 12,
  "button-md": 12,
  "button-lg": 14,
  pill: 9999,
} as const;

/** Layout constants in px. */
export const layout = {
  pagePadding: 24,
  minTouchTarget: 44,
} as const;

/**
 * Font families as registered with expo-font (see lib/fonts.ts). React Native
 * selects a weight by loading one family per weight, so each weight has its own name.
 */
export const fontFamilies = {
  sans: "Geist_400Regular",
  "sans-medium": "Geist_500Medium",
  "sans-semibold": "Geist_600SemiBold",
  "sans-bold": "Geist_700Bold",
  display: "Fraunces_600SemiBold",
  "display-medium": "Fraunces_500Medium",
  "display-bold": "Fraunces_700Bold",
} as const;

/** Custom font-size utilities as [size, lineHeight] in px. */
export const fontSizes = {
  caption: [12, 16],
  secondary: [13, 18],
  "body-sm": [14, 20],
  body: [15, 22],
  title: [26, 32],
  "title-lg": [28, 34],
  "hero-sm": [44, 48],
  hero: [56, 60],
  "hero-lg": [72, 76],
} as const satisfies Record<string, readonly [number, number]>;
