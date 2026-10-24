import { Pressable, View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Text } from "./text";

export type SegmentedItem = {
  value: string;
  /** German label. */
  label: string;
};

export type SegmentedProps = Omit<ViewProps, "children"> & {
  items: readonly SegmentedItem[];
  /** Selected value (controlled). */
  value: string;
  onValueChange: (value: string) => void;
  className?: string;
};

/**
 * Underline switch between a few views of one screen ("Tagebuch", "Gesundheit", "Akte"): labels in
 * a row on a hairline, the selected one with a brass underline. Quieter than `Tabs` (no boxes), for
 * the content below the head of a page. Put the content below it and render it yourself.
 *
 * @example
 * <Segmented items={[{ value: "diary", label: "Tagebuch" }, { value: "file", label: "Akte" }]} value={v} onValueChange={setV} />
 */
export function Segmented({ items, value, onValueChange, className, ...props }: SegmentedProps) {
  return (
    <View accessibilityRole="tablist" className={cn("flex-row border-b border-divider", className)} {...props}>
      {items.map((item) => {
        const selected = item.value === value;
        return (
          <Pressable
            key={item.value}
            accessibilityRole="tab"
            accessibilityState={{ selected }}
            onPress={() => onValueChange(item.value)}
            className={cn(
              "min-h-touch flex-1 items-center justify-center border-b-2 px-2 active:opacity-70",
              selected ? "border-brass" : "border-transparent",
            )}
          >
            <Text
              className={cn("text-body-sm", selected ? "font-sans-semibold text-foreground" : "font-sans-medium text-muted")}
              numberOfLines={1}
            >
              {item.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}
