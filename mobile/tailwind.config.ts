import type { Config } from "tailwindcss";

import { colors, fontFamilies, fontSizes, radii } from "./lib/tokens.ts";

// eslint-disable-next-line @typescript-eslint/no-require-imports
const nativewindPreset = require("nativewind/preset");

// All design tokens live in lib/tokens.ts; this file only maps them to Tailwind.
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
      ...colors,
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
      DEFAULT: colors.border,
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
