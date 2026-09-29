import type { LucideIcon } from "lucide-react-native";
import { createContext, useContext } from "react";
import { Pressable, View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Icon } from "./icon";
import { Text } from "./text";

type ToggleGroupContextValue = {
  isSelected: (value: string) => boolean;
  toggle: (value: string) => void;
};
const ToggleGroupContext = createContext<ToggleGroupContextValue | null>(null);

type BaseProps = Omit<ViewProps, "children"> & {
  children: React.ReactNode;
  className?: string;
};

export type ToggleGroupProps = BaseProps &
  (
    | {
        /** Exactly one item can be selected; tapping the selected item keeps it selected. */
        type: "single";
        value: string;
        onValueChange: (value: string) => void;
      }
    | {
        /** Any number of items can be selected. */
        type: "multiple";
        value: string[];
        onValueChange: (value: string[]) => void;
      }
  );

/**
 * Row of equally wide toggle buttons (shadcn API).
 *
 * @example
 * <ToggleGroup type="single" value={gait} onValueChange={setGait}>
 *   <ToggleGroupItem value="walk" label="Schritt" />
 *   <ToggleGroupItem value="trot" label="Trab" />
 * </ToggleGroup>
 */
export function ToggleGroup(props: ToggleGroupProps) {
  const { className, children, type, value, onValueChange, ...rest } = props as BaseProps & {
    type: "single" | "multiple";
    value: string | string[];
    onValueChange: (value: never) => void;
  };

  const ctx: ToggleGroupContextValue = {
    isSelected: (v) => (type === "single" ? value === v : (value as string[]).includes(v)),
    toggle: (v) => {
      if (type === "single") {
        (onValueChange as (value: string) => void)(v);
      } else {
        const list = value as string[];
        (onValueChange as (value: string[]) => void)(
          list.includes(v) ? list.filter((x) => x !== v) : [...list, v],
        );
      }
    },
  };

  return (
    <ToggleGroupContext.Provider value={ctx}>
      <View accessibilityRole="radiogroup" className={cn("flex-row gap-2", className)} {...rest}>
        {children}
      </View>
    </ToggleGroupContext.Provider>
  );
}

export type ToggleGroupItemProps = {
  value: string;
  /** German label. */
  label: string;
  icon?: LucideIcon;
  disabled?: boolean;
  className?: string;
};

export function ToggleGroupItem({ value, label, icon, disabled, className }: ToggleGroupItemProps) {
  const ctx = useContext(ToggleGroupContext);
  if (!ctx) throw new Error("ToggleGroupItem must be used inside <ToggleGroup>");
  const selected = ctx.isSelected(value);
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected, disabled: !!disabled }}
      disabled={disabled}
      onPress={() => ctx.toggle(value)}
      className={cn(
        "min-h-touch flex-1 flex-row items-center justify-center gap-1.5 rounded-button-md border px-3",
        selected ? "border-primary bg-primary-soft" : "border-border bg-card active:bg-background",
        disabled && "opacity-50",
        className,
      )}
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
