// Design tokens as plain data. This file has no React Native imports so that
// tailwind.config.ts, the app and the unit tests can all share it.
// Documentation: docs/design-system.md at the repo root.

/** Semantic color tokens of one scheme. Keys map 1:1 to Tailwind color names. */
export type Palette = {
  background: string;
  card: string;
  /** Raised surface inside a screen: the next task, a running flow. */
  raised: string;
  border: string;
  divider: string;
  foreground: string;
  muted: string;
  /** The one action of a screen. */
  primary: { DEFAULT: string; foreground: string; soft: string; deep: string; deeper: string };
  /** Times, numbers and small highlights ("brass"). */
  brass: { DEFAULT: string; soft: string };
  /** Done, fine, in good shape. */
  success: { DEFAULT: string; soft: string };
  /** Warnings ("ember"). */
  accent: { DEFAULT: string; soft: string; text: string };
  info: { DEFAULT: string; soft: string };
  /** Overdue, errors, destructive. */
  danger: { DEFAULT: string; soft: string };
  /** The dark readout of a running session (`LivePanel`), dark in both schemes. */
  panel: string;
  /** Gait colors. `canter` is for dark backgrounds, `canter-light` for light ones. */
  gait: { walk: string; trot: string; canter: string; "canter-light": string };
};

const gait = { walk: "#86efac", trot: "#fbbf24", canter: "#ffffff", "canter-light": "#c2410c" };

/**
 * "Abendlicht" (docs/design-system.md): the app follows the phone's light or dark setting.
 * Light is warm paper by day, dark is the stable at night with brass for the one action.
 */
export const palettes: { light: Palette; dark: Palette } = {
  light: {
    background: "#f4f0e6",
    card: "#fffdf8",
    raised: "#fffdf8",
    border: "#e6dfd1",
    divider: "#eee8db",
    foreground: "#1a1814",
    muted: "#6d675c",
    primary: { DEFAULT: "#1f3527", foreground: "#f4f0e6", soft: "#e2e8dc", deep: "#1f3527", deeper: "#14231a" },
    brass: { DEFAULT: "#8a6a2c", soft: "#f1e8d4" },
    success: { DEFAULT: "#2d6a45", soft: "#e1eee3" },
    accent: { DEFAULT: "#c2410c", soft: "#fbecdf", text: "#9a4d0b" },
    info: { DEFAULT: "#1e4f8a", soft: "#e6eefb" },
    danger: { DEFAULT: "#9f1d1d", soft: "#fbe4e2" },
    panel: "#16241b",
    gait,
  },
  dark: {
    background: "#0e1210",
    card: "#141a16",
    raised: "#19211c",
    border: "#2a342d",
    divider: "#1f2722",
    foreground: "#ede7da",
    muted: "#98a294",
    primary: { DEFAULT: "#c9a45c", foreground: "#1a140a", soft: "#28261b", deep: "#e8cf98", deeper: "#f3e3bd" },
    brass: { DEFAULT: "#c9a45c", soft: "#28261b" },
    success: { DEFAULT: "#8cc49e", soft: "#1b2a20" },
    accent: { DEFAULT: "#e59457", soft: "#2b1f16", text: "#f2b386" },
    info: { DEFAULT: "#8fb3e6", soft: "#172130" },
    danger: { DEFAULT: "#e48585", soft: "#2e1a1a" },
    panel: "#19241d",
    gait,
  },
};

/** The light palette, for places that cannot follow the scheme (Tailwind config, tests, static assets). */
export const colors = palettes.light;

/** Flat token names ("primary-soft", "background", ...) with their value, for CSS variables. */
export function flattenPalette(palette: Palette): [string, string][] {
  return Object.entries(palette).flatMap(([name, value]) =>
    typeof value === "string"
      ? [[name, value] as [string, string]]
      : Object.entries(value).map(([k, v]) => [k === "DEFAULT" ? name : `${name}-${k}`, v] as [string, string]),
  );
}

/** "#c9a45c" -> "201 164 92" (the format of the CSS variables, so Tailwind opacity modifiers work). */
export function hexToRgbChannels(hex: string): string {
  const n = parseInt(hex.slice(1), 16);
  return `${(n >> 16) & 255} ${(n >> 8) & 255} ${n & 255}`;
}

/**
 * The CSS variables of one palette, `--color-<token>` to "r g b", exactly what global.css declares
 * for that scheme. `ThemeScope` hands them to NativeWind's `vars()` to pin a subtree to a scheme.
 */
export function paletteVariables(palette: Palette): Record<string, string> {
  return Object.fromEntries(flattenPalette(palette).map(([name, hex]) => [`--color-${name}`, hexToRgbChannels(hex)]));
}

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
  /** Light italic Fraunces: titles with a voice (page titles, the greeting, horse names). */
  "display-italic": "Fraunces_300Light_Italic",
  "display-italic-medium": "Fraunces_500Medium_Italic",
} as const;

/** Custom font-size utilities as [size, lineHeight] in px. */
export const fontSizes = {
  caption: [12, 16],
  secondary: [13, 18],
  "body-sm": [14, 20],
  body: [15, 22],
  heading: [20, 26],
  title: [26, 32],
  "title-lg": [28, 34],
  "title-xl": [34, 38],
  "hero-sm": [44, 48],
  hero: [56, 60],
  "hero-lg": [72, 76],
} as const satisfies Record<string, readonly [number, number]>;
