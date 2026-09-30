import { View } from "react-native";

import { Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import {
  dotViewModels,
  segmentViewModels,
  type ApiDot,
  type ApiSegment,
  type DotTone,
  type Intensity,
} from "@/lib/training";

const DOT_CLASS: Record<DotTone, string> = {
  empty: "border-border bg-card",
  rest: "border-divider bg-divider",
  light: "border-primary bg-primary-soft",
  medium: "border-primary bg-primary",
  intense: "border-primary-deeper bg-primary-deeper",
};

/** Seven dots (last seven days, today last): trained (green by intensity), rest, nothing. */
export function WeekDots({ dots }: { dots: readonly ApiDot[] }) {
  const items = dotViewModels(dots);
  return (
    <View className="flex-row justify-between" accessibilityRole="summary">
      {items.map((d) => (
        <View key={d.key} accessible accessibilityLabel={d.accessibilityLabel} className="items-center gap-1.5">
          <View className={cn("h-7 w-7 rounded-pill border-2", DOT_CLASS[d.tone], d.isToday && "border-accent")} />
          <Text variant="caption" className={d.isToday ? "font-sans-semibold text-foreground" : undefined}>
            {d.label}
          </Text>
        </View>
      ))}
    </View>
  );
}

const SEGMENT_CLASS: Record<Intensity, string> = {
  none: "bg-border",
  light: "bg-primary-soft border border-primary",
  medium: "bg-primary",
  intense: "bg-primary-deeper",
};

const BAR_HEIGHT = 56;

/** Seven bars of the week's training load (Mo to So). */
export function LoadBar({ segments }: { segments: readonly ApiSegment[] }) {
  const items = segmentViewModels(segments);
  return (
    <View className="flex-row items-end justify-between gap-2">
      {items.map((s) => (
        <View key={s.key} accessible accessibilityLabel={s.accessibilityLabel} className="flex-1 items-center gap-1.5">
          <View className="w-full justify-end" style={{ height: BAR_HEIGHT }}>
            <View
              className={cn("w-full rounded-button-sm", SEGMENT_CLASS[s.level])}
              style={{ height: Math.max(4, Math.round(BAR_HEIGHT * s.ratio)) }}
            />
          </View>
          <Text variant="caption">{s.label}</Text>
        </View>
      ))}
    </View>
  );
}
