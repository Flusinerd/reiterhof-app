import type { LucideIcon } from "lucide-react-native";
import { cva, type VariantProps } from "class-variance-authority";
import { ActivityIndicator, Pressable, type PressableProps } from "react-native";

import { cn } from "@/lib/cn";
import { colors } from "@/lib/theme";

import { Icon } from "./icon";
import { Text } from "./text";

const buttonVariants = cva("flex-row items-center justify-center gap-2 border", {
  variants: {
    variant: {
      primary: "border-primary bg-primary active:bg-primary-deep",
      secondary: "border-primary-soft bg-primary-soft active:bg-border",
      outline: "border-border bg-card active:bg-background",
      ghost: "border-transparent bg-transparent active:bg-divider",
      danger: "border-danger bg-danger active:opacity-90",
    },
    size: {
      /** 44 px high, radius 12. */
      sm: "min-h-touch rounded-button-sm px-4",
      /** 48 px high, radius 12. */
      md: "h-12 rounded-button-md px-5",
      /** 56 px high, radius 14. */
      lg: "h-14 rounded-button-lg px-6",
      /** 44 x 44 square, radius 12. */
      icon: "h-11 w-11 rounded-button-sm",
    },
    fullWidth: { true: "self-stretch", false: "self-start" },
    disabled: { true: "opacity-50", false: "" },
  },
  defaultVariants: { variant: "primary", size: "md", fullWidth: false, disabled: false },
});

const buttonTextVariants = cva("font-sans-semibold", {
  variants: {
    variant: {
      primary: "text-white",
      secondary: "text-primary-deep",
      outline: "text-foreground",
      ghost: "text-foreground",
      danger: "text-white",
    },
    size: { sm: "text-body-sm", md: "text-body", lg: "text-body", icon: "text-body" },
  },
  defaultVariants: { variant: "primary", size: "md" },
});

const iconColorClass = {
  primary: "text-white",
  secondary: "text-primary-deep",
  outline: "text-foreground",
  ghost: "text-foreground",
  danger: "text-white",
} as const;

const spinnerColor = {
  primary: colors.card,
  secondary: colors.primary.deep,
  outline: colors.foreground,
  ghost: colors.foreground,
  danger: colors.card,
} as const;

export type ButtonVariant = NonNullable<VariantProps<typeof buttonVariants>["variant"]>;
export type ButtonSize = NonNullable<VariantProps<typeof buttonVariants>["size"]>;

export type ButtonProps = Omit<PressableProps, "children" | "disabled"> & {
  /** German label. Optional only for `size="icon"`, where `accessibilityLabel` is required. */
  label?: string;
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Lucide icon shown before the label (or alone for `size="icon"`). */
  icon?: LucideIcon;
  /** Shows a spinner and blocks presses. */
  loading?: boolean;
  disabled?: boolean;
  /** Stretch to the container width. */
  fullWidth?: boolean;
  className?: string;
};

/**
 * @example
 * <Button label="Anfrage senden" onPress={send} />
 * <Button variant="outline" size="icon" icon={Plus} accessibilityLabel="Hinzufügen" />
 */
export function Button({
  label,
  variant = "primary",
  size = "md",
  icon,
  loading = false,
  disabled = false,
  fullWidth = false,
  className,
  ...props
}: ButtonProps) {
  const isDisabled = disabled || loading;
  const iconSize = size === "sm" ? 16 : 20;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: isDisabled, busy: loading }}
      disabled={isDisabled}
      className={cn(buttonVariants({ variant, size, fullWidth, disabled: isDisabled }), className)}
      {...props}
    >
      {loading ? (
        <ActivityIndicator size="small" color={spinnerColor[variant]} />
      ) : icon ? (
        <Icon as={icon} size={iconSize} className={iconColorClass[variant]} />
      ) : null}
      {label && size !== "icon" ? (
        <Text className={buttonTextVariants({ variant, size })}>{label}</Text>
      ) : null}
    </Pressable>
  );
}
