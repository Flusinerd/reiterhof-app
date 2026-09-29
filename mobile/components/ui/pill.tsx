import type { LucideIcon } from "lucide-react-native";
import { Pressable, type PressableProps } from "react-native";

import { cn } from "@/lib/cn";

import { Icon } from "./icon";
import { Text } from "./text";

export type PillProps = Omit<PressableProps, "children"> & {
  /** German label. */
  label: string;
  /** Selected state, e.g. the active filter. */
  selected?: boolean;
  icon?: LucideIcon;
  className?: string;
};

/**
 * Tappable rounded chip for filters and quick choices. 36 px visual height with a
 * 44 px touch target (via `hitSlop`). Use `Badge` for non-interactive status labels.
 *
 * @example <Pill label="Alle" selected={filter === "all"} onPress={() => setFilter("all")} />
 */
export function Pill({ label, selected = false, icon, className, ...props }: PillProps) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected }}
      hitSlop={{ top: 4, bottom: 4, left: 2, right: 2 }}
      className={cn(
        "h-9 flex-row items-center justify-center gap-1.5 self-start rounded-pill border px-4",
        selected
          ? "border-primary bg-primary-soft"
          : "border-border bg-card active:bg-background",
        className,
      )}
      {...props}
    >
      {icon ? (
        <Icon as={icon} size={16} className={selected ? "text-primary-deep" : "text-muted"} />
      ) : null}
      <Text
        className={cn(
          "text-body-sm",
          selected ? "font-sans-semibold text-primary-deep" : "font-sans-medium text-foreground",
        )}
      >
        {label}
      </Text>
    </Pressable>
  );
}
