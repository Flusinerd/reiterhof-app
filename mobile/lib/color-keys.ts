// Named color keys for horses and people. The backend stores one `color_key`
// per horse/user; the app maps it to an avatar background/foreground pair.

export const COLOR_KEYS = [
  "green",
  "amber",
  "blue",
  "rose",
  "violet",
  "teal",
  "neutral",
] as const;

export type ColorKey = (typeof COLOR_KEYS)[number];

export type ColorPair = { readonly bg: string; readonly fg: string };

export const COLOR_PAIRS: Readonly<Record<ColorKey, ColorPair>> = {
  green: { bg: "#a3c9ad", fg: "#1c3a27" },
  amber: { bg: "#f0c380", fg: "#4a2a05" },
  blue: { bg: "#b9c7ea", fg: "#1e3560" },
  rose: { bg: "#f2b8c6", fg: "#5a1a2c" },
  violet: { bg: "#cbb8e8", fg: "#3b1f63" },
  teal: { bg: "#9fd6cf", fg: "#0f3f3a" },
  neutral: { bg: "#e7e2d9", fg: "#44403c" },
};

/**
 * The same keys for the dark scheme ("Abendlicht"): the light pastels would glow on the near-black
 * background, so the field is a muted, dark tint and the initial a light version of the hue.
 */
export const DARK_COLOR_PAIRS: Readonly<Record<ColorKey, ColorPair>> = {
  green: { bg: "#2c4434", fg: "#b9dcc3" },
  amber: { bg: "#4a3b22", fg: "#e8cf98" },
  blue: { bg: "#2b3550", fg: "#bccaee" },
  rose: { bg: "#4a2d37", fg: "#f0bccb" },
  violet: { bg: "#3d3254", fg: "#d2c3ee" },
  teal: { bg: "#254440", fg: "#a9ddd5" },
  neutral: { bg: "#262e28", fg: "#b3bcae" },
};

export type ColorSchemeName = "light" | "dark";

export function isColorKey(value: unknown): value is ColorKey {
  return typeof value === "string" && (COLOR_KEYS as readonly string[]).includes(value);
}

/** Returns a valid key; unknown, empty or missing values fall back to `neutral`. */
export function resolveColorKey(value: string | null | undefined): ColorKey {
  const normalized = value?.trim().toLowerCase();
  return isColorKey(normalized) ? normalized : "neutral";
}

/** Background/foreground colors for a (possibly unknown) color key; light unless `scheme` is "dark". */
export function colorsForKey(value: string | null | undefined, scheme: ColorSchemeName = "light"): ColorPair {
  return (scheme === "dark" ? DARK_COLOR_PAIRS : COLOR_PAIRS)[resolveColorKey(value)];
}
