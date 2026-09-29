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

export function isColorKey(value: unknown): value is ColorKey {
  return typeof value === "string" && (COLOR_KEYS as readonly string[]).includes(value);
}

/** Returns a valid key; unknown, empty or missing values fall back to `neutral`. */
export function resolveColorKey(value: string | null | undefined): ColorKey {
  const normalized = value?.trim().toLowerCase();
  return isColorKey(normalized) ? normalized : "neutral";
}

/** Background/foreground colors for a (possibly unknown) color key. */
export function colorsForKey(value: string | null | undefined): ColorPair {
  return COLOR_PAIRS[resolveColorKey(value)];
}
