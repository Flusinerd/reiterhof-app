import { View } from "react-native";

import { Text } from "@/components/ui";
import type { Gait } from "@/lib/gait";
import { colors } from "@/lib/theme";
import type { GaitPoint } from "@/lib/tracking";

export const MAP_GAIT_COLORS: Record<Gait, string> = {
  halt: colors.muted,
  walk: colors.gait.walk,
  trot: colors.gait.trot,
  canter: colors.gait["canter-light"],
};

/** Web has no native map; the tracker is a phone feature. */
export function TrackingMap({ height = 280 }: { points: GaitPoint[]; height?: number; follow?: boolean }) {
  return (
    <View style={{ height }} className="items-center justify-center rounded-card border border-border bg-card">
      <Text variant="secondary">Die Karte gibt es nur in der App.</Text>
    </View>
  );
}
