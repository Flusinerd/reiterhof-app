import { cva, type VariantProps } from "class-variance-authority";
import { View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Text } from "./text";

const badgeVariants = cva("flex-row items-center self-start rounded-pill px-2.5 py-1", {
  variants: {
    variant: {
      neutral: "bg-divider",
      primary: "bg-primary-soft",
      accent: "bg-accent-soft",
      info: "bg-info-soft",
      danger: "bg-danger-soft",
    },
  },
  defaultVariants: { variant: "neutral" },
});

const badgeTextVariants = cva("font-sans-semibold text-caption", {
  variants: {
    variant: {
      neutral: "text-muted",
      primary: "text-primary-deep",
      accent: "text-accent-text",
      info: "text-info",
      danger: "text-danger",
    },
  },
  defaultVariants: { variant: "neutral" },
});

export type BadgeProps = Omit<ViewProps, "children"> &
  VariantProps<typeof badgeVariants> & {
    /** German label, e.g. "Offen". */
    label: string;
    className?: string;
  };

/**
 * Small static status label (not tappable; use `Pill` for filters).
 *
 * @example <Badge variant="accent" label="Dringend" />
 */
export function Badge({ label, variant, className, ...props }: BadgeProps) {
  return (
    <View className={cn(badgeVariants({ variant }), className)} {...props}>
      <Text className={badgeTextVariants({ variant })}>{label}</Text>
    </View>
  );
}
