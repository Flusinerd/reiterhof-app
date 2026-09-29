import { cva, type VariantProps } from "class-variance-authority";
import type { ReactNode } from "react";
import { View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Text, TextToneContext, type TextTone } from "./text";

const heroVariants = cva("gap-3 rounded-card border p-6", {
  variants: {
    /**
     * `forest`: dark green (gait colors are designed for this), `deep`: darkest green,
     * `soft`: light green, `warm`: light orange, `plain`: white card.
     */
    tone: {
      forest: "border-primary-deep bg-primary-deep",
      deep: "border-primary-deeper bg-primary-deeper",
      soft: "border-primary-soft bg-primary-soft",
      warm: "border-accent-soft bg-accent-soft",
      plain: "border-border bg-card",
    },
  },
  defaultVariants: { tone: "forest" },
});

type HeroTone = NonNullable<VariantProps<typeof heroVariants>["tone"]>;

const textTone: Record<HeroTone, TextTone> = {
  forest: "inverse",
  deep: "inverse",
  soft: "default",
  warm: "default",
  plain: "default",
};

export type HeroProps = Omit<ViewProps, "children"> & {
  tone?: HeroTone;
  /** Small line above the title/value, e.g. "Heute, 14:30". */
  eyebrow?: string;
  /** Page title (Fraunces 26 px). Give a `title`, a `value`, or both. */
  title?: string;
  /** Big number (Fraunces), e.g. "12" or "18°". */
  value?: string;
  /** Size of the big number: 44, 56 or 72 px. */
  valueSize?: "sm" | "md" | "lg";
  /** Unit or caption next to the big number, e.g. "Grad". */
  unit?: string;
  /** Supporting text below. */
  description?: string;
  /** Extra content below the description (buttons, gait row, ...). */
  children?: ReactNode;
  className?: string;
};

const valueVariant = { sm: "heroNumberSm", md: "heroNumber", lg: "heroNumberLg" } as const;

/**
 * The single focal element at the top of every screen: exactly one `Hero` per
 * screen. Children inherit the text tone (white on the dark tones).
 *
 * @example
 * <Hero eyebrow="Heute" title="Guten Morgen" description="Luna braucht eine Decke." />
 * <Hero tone="soft" value="7°" unit="gefühlt" description="Leichte Decke empfohlen" />
 */
export function Hero({
  tone = "forest",
  eyebrow,
  title,
  value,
  valueSize = "md",
  unit,
  description,
  children,
  className,
  ...props
}: HeroProps) {
  const dark = tone === "forest" || tone === "deep";
  return (
    <TextToneContext.Provider value={textTone[tone]}>
      <View className={cn(heroVariants({ tone }), className)} {...props}>
        {eyebrow ? (
          <Text variant="secondary" className={dark ? "text-white/70" : undefined}>
            {eyebrow}
          </Text>
        ) : null}
        {title ? (
          <Text variant="title" accessibilityRole="header">
            {title}
          </Text>
        ) : null}
        {value ? (
          <View className="flex-row items-baseline gap-2">
            <Text variant={valueVariant[valueSize]}>{value}</Text>
            {unit ? (
              <Text variant="body" className={dark ? "text-white/70" : "text-muted"}>
                {unit}
              </Text>
            ) : null}
          </View>
        ) : null}
        {description ? (
          <Text variant="body" className={dark ? "text-white/85" : "text-muted"}>
            {description}
          </Text>
        ) : null}
        {children}
      </View>
    </TextToneContext.Provider>
  );
}
