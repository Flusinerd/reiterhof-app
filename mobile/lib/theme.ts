// Values that are needed in JS (icons, navigation options, native props) and
// cannot be expressed as class names. Everything is derived from lib/tokens.ts.
import { colors, layout } from "./tokens.ts";

export { colors, layout } from "./tokens.ts";

export const icon = {
  /** Lucide stroke width. */
  strokeWidth: 2,
  sizes: { sm: 16, md: 20, lg: 24 },
} as const;

export const tabBar = {
  height: 68,
  paddingTop: 8,
  paddingBottom: 8,
  activeTint: colors.primary.DEFAULT,
  inactiveTint: colors.muted,
  background: colors.card,
  borderColor: colors.border,
  /** Size of the pill behind the active icon. */
  pill: { width: 56, height: 32 },
  iconSize: icon.sizes.md,
  labelFont: "Geist_500Medium",
  labelSize: 12,
} as const;

export const switchColors = {
  trackOff: colors.border,
  trackOn: colors.primary.DEFAULT,
  thumb: colors.card,
} as const;

export const sheet = {
  backdrop: "rgba(28, 25, 23, 0.4)",
} as const;

export const minTouchTarget = layout.minTouchTarget;
