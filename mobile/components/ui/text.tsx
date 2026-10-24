import { cva, type VariantProps } from "class-variance-authority";
import { createContext, useContext } from "react";
import { Text as RNText, type TextProps as RNTextProps } from "react-native";

import { cn } from "@/lib/cn";

export const textVariants = cva("text-foreground", {
  variants: {
    variant: {
      /** Page title at the top of a screen, Fraunces light italic 34 px. Use through `PageHeader`. */
      display: "font-display-italic text-title-xl",
      /** Section heading, Fraunces medium italic 20 px. Use through `SectionTitle`. */
      heading: "font-display-italic-medium text-heading",
      /** Title of a sheet or a prominent block, Fraunces light italic 26 px. */
      title: "font-display-italic text-title",
      /** Larger title, Fraunces light italic 28 px. */
      titleLg: "font-display-italic text-title-lg",
      /** Big number, Fraunces 44 px. Use inside a `Hero`. */
      heroNumberSm: "font-display text-hero-sm",
      /** Big number, Fraunces 56 px. */
      heroNumber: "font-display text-hero",
      /** Big number, Fraunces 72 px. */
      heroNumberLg: "font-display text-hero-lg",
      /** Body text, Geist 15 px. */
      body: "font-sans text-body",
      /** Body text, Geist 14 px. */
      bodySm: "font-sans text-body-sm",
      /** Emphasized body text, Geist 15 px semibold. */
      bodyStrong: "font-sans-semibold text-body",
      /** Secondary text, Geist 13 px muted. */
      secondary: "font-sans text-secondary text-muted",
      /** Small caption, Geist 12 px muted. */
      caption: "font-sans text-caption text-muted",
      /** Form field label, Geist 13 px / 600 muted. */
      label: "font-sans-semibold text-secondary text-muted",
      /** Small line above a title or row: Geist 11 px / 600, upper case, spaced, muted. */
      eyebrow: "font-sans-semibold text-[11px] uppercase leading-4 tracking-[1.4px] text-muted",
    },
    tone: {
      default: "",
      muted: "text-muted",
      primary: "text-primary",
      /** Times, numbers, small highlights. */
      brass: "text-brass",
      success: "text-success",
      accent: "text-accent-text",
      info: "text-info",
      danger: "text-danger",
      /** White text, for dark surfaces like `Hero`. */
      inverse: "text-white",
    },
  },
  defaultVariants: { variant: "body", tone: "default" },
});

export type TextVariant = NonNullable<VariantProps<typeof textVariants>["variant"]>;
export type TextTone = NonNullable<VariantProps<typeof textVariants>["tone"]>;

/** Lets a parent (e.g. `Hero`) set a default tone for nested `Text`s. */
export const TextToneContext = createContext<TextTone | undefined>(undefined);

export type TextProps = RNTextProps & VariantProps<typeof textVariants> & {
  className?: string;
};

/**
 * The only text primitive of the kit. Fonts are applied through the variant, so
 * never set `fontWeight` yourself: React Native needs one font family per weight.
 *
 * @example <Text variant="title">Start</Text>
 */
export function Text({ className, variant, tone, ...props }: TextProps) {
  const inheritedTone = useContext(TextToneContext);
  return (
    <RNText
      className={cn(textVariants({ variant, tone: tone ?? inheritedTone }), className)}
      {...props}
    />
  );
}
