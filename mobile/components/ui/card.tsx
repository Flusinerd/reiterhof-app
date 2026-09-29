import { cva, type VariantProps } from "class-variance-authority";
import { Pressable, View, type PressableProps, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

const cardVariants = cva("border border-border bg-card", {
  variants: {
    /** `card`: radius 20 (default). `tile`: radius 16, for small items inside a card or grids. */
    shape: { card: "rounded-card", tile: "rounded-tile" },
    padded: { true: "p-5", false: "" },
  },
  defaultVariants: { shape: "card", padded: true },
});

export type CardProps = ViewProps &
  VariantProps<typeof cardVariants> & { className?: string };

/**
 * White surface with 1 px border, no shadow. Padded (20 px) by default.
 *
 * @example
 * <Card><Text variant="bodyStrong">Luna</Text></Card>
 */
export function Card({ className, shape, padded, ...props }: CardProps) {
  return <View className={cn(cardVariants({ shape, padded }), className)} {...props} />;
}

export type PressableCardProps = PressableProps &
  VariantProps<typeof cardVariants> & { className?: string };

/** A `Card` that can be tapped (list rows that navigate somewhere). */
export function PressableCard({ className, shape, padded, ...props }: PressableCardProps) {
  return (
    <Pressable
      accessibilityRole="button"
      className={cn(cardVariants({ shape, padded }), "active:bg-background", className)}
      {...props}
    />
  );
}

/** 1 px hairline used between rows inside a card. */
export function Divider({ className, ...props }: ViewProps & { className?: string }) {
  return <View className={cn("h-px bg-divider", className)} {...props} />;
}
