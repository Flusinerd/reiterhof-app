import { Check, Moon } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { ActivityIcon } from "@/components/training-activity-icon";
import { Icon, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import { weekStripItems, type WeekDayLike, type WeekStripItem } from "@/lib/training";

const CIRCLE_CLASS: Record<WeekStripItem["tone"], string> = {
  done: "bg-primary",
  planned: "border border-primary bg-primary-soft",
  rest: "bg-divider",
  open: "border border-border bg-card",
  empty: "bg-divider",
};

function CircleContent({ item }: { item: WeekStripItem }) {
  if (item.tone === "done") return <Icon as={Check} size={16} className="text-white" />;
  if (item.tone === "planned" && item.iconKey && item.iconKey !== "check") {
    return <ActivityIcon activity={item.iconKey} size={16} />;
  }
  if (item.tone === "rest") return <Icon as={Moon} size={16} className="text-muted" />;
  return null;
}

export type WeekStripProps = {
  days: readonly WeekDayLike[];
  onPress: () => void;
};

/** The seven days of the current week as a tappable strip: done, planned, rest, open; today outlined. */
export function WeekStrip({ days, onPress }: WeekStripProps) {
  const items = weekStripItems(days);
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel="Woche öffnen"
      accessibilityHint={items.map((i) => i.accessibilityLabel).join("; ")}
      onPress={onPress}
      className="min-h-touch flex-row gap-1"
    >
      {items.map((item) => (
        <View key={item.key} className="flex-1 items-center gap-1">
          <Text variant="caption" className={item.isToday ? "text-accent-text" : undefined}>
            {item.label}
          </Text>
          <View
            className={cn(
              "h-9 w-9 items-center justify-center rounded-pill",
              CIRCLE_CLASS[item.tone],
              item.isToday && "border-2 border-accent",
            )}
          >
            <CircleContent item={item} />
          </View>
        </View>
      ))}
    </Pressable>
  );
}
