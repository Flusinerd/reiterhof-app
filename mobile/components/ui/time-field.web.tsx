import { X } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { cn } from "@/lib/cn";

import { Icon } from "./icon";
import type { TimeFieldProps } from "./time-field";

/**
 * Web variant of `TimeField`: the browser's own `<input type="time">`, styled like `Input`
 * (`@react-native-community/datetimepicker` has no web implementation).
 */
export function TimeField({ value, onChange, accessibilityLabel, clearable = false, minuteInterval, className }: TimeFieldProps) {
  return (
    <View className={cn("h-12 flex-row items-center rounded-button-md border border-border bg-card", className)}>
      <input
        type="time"
        value={value}
        step={minuteInterval ? minuteInterval * 60 : undefined}
        aria-label={accessibilityLabel}
        onChange={(e) => onChange(e.target.value)}
        className="h-full min-w-0 flex-1 bg-transparent px-4 font-sans text-body text-foreground outline-none"
        style={{ font: "inherit", border: 0, minWidth: 0 }}
      />
      {clearable && value ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${accessibilityLabel} entfernen`}
          hitSlop={8}
          onPress={() => onChange("")}
          className="h-full items-center justify-center px-3 active:opacity-70"
        >
          <Icon as={X} size={16} className="text-muted" />
        </Pressable>
      ) : null}
    </View>
  );
}
