import { Check } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { Icon, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import type { StripItem, StripTone } from "@/lib/horse-page";

const MARK_CLASS: Record<StripTone, string> = {
  done: "border-success/40 bg-success/25",
  planned: "border-dashed border-muted/60",
  show: "border-brass bg-brass/35",
  rest: "border-divider bg-divider",
  empty: "border-divider bg-card",
};

/**
 * Seven days as a compact strip: done (green), planned (dashed), show day (brass), rest and empty
 * days quiet; today's weekday in brass. Tapping opens the week.
 */
export function HorseWeekStrip({ items, onPress }: { items: readonly StripItem[]; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel="Woche öffnen"
      accessibilityHint={items.map((i) => i.accessibilityLabel).join("; ")}
      onPress={onPress}
      className="min-h-touch flex-row gap-1 active:opacity-70"
    >
      {items.map((item) => (
        <View key={item.key} className="flex-1 items-center gap-1.5">
          <View className={cn("h-7 w-full items-center justify-center rounded-button-sm border", MARK_CLASS[item.tone])}>
            {item.tone === "done" ? <Icon as={Check} size={14} className="text-success" /> : null}
          </View>
          <Text
            variant="caption"
            className={item.isToday ? "font-sans-semibold text-brass" : undefined}
          >
            {item.label}
          </Text>
        </View>
      ))}
    </Pressable>
  );
}
