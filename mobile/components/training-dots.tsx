import { View } from "react-native";

import { Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import { segmentViewModels, type ApiSegment, type Intensity } from "@/lib/training";

const SEGMENT_CLASS: Record<Intensity, string> = {
  none: "bg-border",
  light: "bg-primary-soft border border-primary",
  medium: "bg-primary",
  intense: "bg-primary-deeper",
};

const BAR_HEIGHT = 48;

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
