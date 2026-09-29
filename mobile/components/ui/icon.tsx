import type { LucideIcon, LucideProps } from "lucide-react-native";
import { cssInterop } from "nativewind";

import { icon as iconTokens } from "@/lib/theme";

// Lucide icons take `color`/`size` props, not styles. `cssInterop` lets us style
// them with `className` (e.g. `text-primary`) like everything else.
const interopped = new WeakSet<object>();

function withClassName(Component: LucideIcon): LucideIcon {
  if (!interopped.has(Component)) {
    cssInterop(Component, {
      className: {
        target: "style",
        nativeStyleToProp: { color: true, opacity: true },
      },
    });
    interopped.add(Component);
  }
  return Component;
}

export type IconProps = Omit<LucideProps, "ref"> & {
  /** A Lucide icon component, e.g. `Home` from `lucide-react-native`. */
  as: LucideIcon;
  /** Tailwind classes; use a `text-*` class for the color. */
  className?: string;
};

/**
 * Lucide icon with the kit defaults (24 px box, stroke 2) and `className` support.
 *
 * @example <Icon as={Check} size={20} className="text-primary" />
 */
export function Icon({
  as,
  size = iconTokens.sizes.lg,
  strokeWidth = iconTokens.strokeWidth,
  ...props
}: IconProps) {
  const Component = withClassName(as);
  return <Component size={size} strokeWidth={strokeWidth} {...props} />;
}
