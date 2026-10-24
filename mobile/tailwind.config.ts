import type { Config } from "tailwindcss";

import { colors, fontFamilies, fontSizes, radii } from "./lib/tokens.ts";

// eslint-disable-next-line @typescript-eslint/no-require-imports
const nativewindPreset = require("nativewind/preset");

// All design tokens live in lib/tokens.ts; this file only maps them to Tailwind. Colors point at
// the CSS variables of global.css, which switch with the phone's light/dark setting.
const themed = (name: string) => `rgb(var(--color-${name}) / <alpha-value>)`;
const colorTokens: Record<string, string | Record<string, string>> = {};
for (const [name, value] of Object.entries(colors)) {
  colorTokens[name] =
    typeof value === "string"
      ? themed(name)
      : Object.fromEntries(Object.keys(value).map((k) => [k, themed(k === "DEFAULT" ? name : `${name}-${k}`)]));
}
const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  presets: [nativewindPreset],
  theme: {
    // Deliberately replaces (not extends) the default palette so that no zinc/gray
    // utilities are available to reach for.
    colors: {
      transparent: "transparent",
      current: "currentColor",
      white: "#ffffff",
      ...colorTokens,
    },
    fontFamily: Object.fromEntries(
      Object.entries(fontFamilies).map(([name, family]) => [name, [family]]),
    ),
    fontSize: {
      ...Object.fromEntries(
        Object.entries(fontSizes).map(([name, [size, lineHeight]]) => [
          name,
          [`${size}px`, `${lineHeight}px`],
        ]),
      ),
      // Keep the small default steps for icon-adjacent text.
      xs: ["12px", "16px"],
      sm: ["14px", "20px"],
      base: ["16px", "24px"],
    },
    borderRadius: {
      none: "0px",
      sm: "8px",
      DEFAULT: "12px",
      md: "12px",
      lg: "14px",
      xl: "16px",
      "2xl": "20px",
      full: "9999px",
      ...Object.fromEntries(
        Object.entries(radii).map(([name, value]) => [name, `${value}px`]),
      ),
    },
    borderColor: ({ theme }) => ({
      ...theme("colors"),
      DEFAULT: themed("border"),
    }),
    extend: {
      spacing: { page: "24px", touch: "44px" },
      minHeight: { touch: "44px" },
      minWidth: { touch: "44px" },
    },
  },
  plugins: [],
};

export default config;
