import { clsx, type ClassValue } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

import { fontSizes } from "./tokens.ts";

// The custom font-size utilities (text-title, text-hero, ...) must be known to
// tailwind-merge, otherwise it would treat them as text colors and drop one of
// `text-title text-foreground`.
const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      "font-size": [{ text: Object.keys(fontSizes) }],
    },
  },
});

/** Joins class names and resolves conflicting Tailwind utilities (shadcn-style). */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
