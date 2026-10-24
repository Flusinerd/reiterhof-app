// Values that are needed in JS (icons, navigation options, native props) and
// cannot be expressed as class names. Everything is derived from lib/tokens.ts.
import { Appearance, useColorScheme } from "react-native";

import { layout, palettes, type Palette } from "./tokens.ts";

export { layout, palettes, type Palette } from "./tokens.ts";

/** The palette of the phone's current light/dark setting; re-renders when it changes. */
export function useColors(): Palette {
  return useColorScheme() === "dark" ? palettes.dark : palettes.light;
}

/**
 * The current palette for plain values (spinner and refresh tints, map markers). It is read when a
 * component renders, so it follows the scheme from the next render on. Screen backgrounds and the
 * tab bar use `useColors()` so that they switch at once.
 */
export const colors: Palette = Object.defineProperties(
  {} as Palette,
  Object.fromEntries(
    (Object.keys(palettes.light) as (keyof Palette)[]).map((key) => [
      key,
      { enumerable: true, get: () => palettes[Appearance.getColorScheme() === "dark" ? "dark" : "light"][key] },
    ]),
  ),
);

export const icon = {
  /** Lucide stroke width. */
  strokeWidth: 2,
  sizes: { sm: 16, md: 20, lg: 24 },
} as const;

/** Dimensions of the bottom tab bar; its colors come from `useColors()`. */
export const tabBar = {
  height: 72,
  paddingTop: 8,
  paddingBottom: 10,
  iconSize: icon.sizes.md,
  labelFont: "Geist_500Medium",
  labelSize: 11,
} as const;

export function switchColors(c: Palette) {
  return { trackOff: c.border, trackOn: c.success.DEFAULT, thumb: c.card } as const;
}

export const sheet = {
  backdrop: "rgba(8, 10, 9, 0.55)",
} as const;

export const minTouchTarget = layout.minTouchTarget;
