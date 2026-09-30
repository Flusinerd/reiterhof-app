import { useRouter } from "expo-router";
import { Pause, Play, Square } from "lucide-react-native";
import { useEffect } from "react";
import { Alert, BackHandler, Platform, View } from "react-native";

import { MAP_GAIT_COLORS } from "@/components/tracking-map";
import { Button, Card, Text } from "@/components/ui";
import type { Gait } from "@/lib/gait";
import { colors } from "@/lib/theme";
import { gaitLabel } from "@/lib/tracking-format";

/** Current gait as a colored dot and a word, for the dark hero (canter is white there). */
export function GaitChip({ gait }: { gait: Gait | null }) {
  const color = gait === null ? colors.muted : gait === "halt" ? "#d6d3d1" : colors.gait[gait];
  return (
    <View className="flex-row items-center gap-2" accessibilityLabel={gait ? `Gangart: ${gaitLabel(gait)}` : "Gangart wird erkannt"}>
      <View className="h-3 w-3 rounded-pill" style={{ backgroundColor: color }} />
      <Text variant="bodyStrong" tone="inverse">
        {gait ? gaitLabel(gait) : "Erkennt …"}
      </Text>
    </View>
  );
}

/** Legend below the map: line colors by gait. */
export function GaitLegend() {
  const gaits: Gait[] = ["walk", "trot", "canter", "halt"];
  return (
    <View className="flex-row flex-wrap gap-x-4 gap-y-1">
      {gaits.map((g) => (
        <View key={g} className="flex-row items-center gap-1.5">
          <View className="h-1.5 w-5 rounded-pill" style={{ backgroundColor: MAP_GAIT_COLORS[g] }} />
          <Text variant="caption">{gaitLabel(g)}</Text>
        </View>
      ))}
    </View>
  );
}

export function StatTile({ label, value }: { label: string; value: string }) {
  return (
    <Card shape="tile" className="flex-1 gap-1 p-4" accessibilityLabel={`${label}: ${value}`}>
      <Text variant="caption">{label}</Text>
      <Text variant="bodyStrong">{value}</Text>
    </Card>
  );
}

/** Thin progress bar (0..1) in the hero. */
export function ProgressBar({ value }: { value: number }) {
  return (
    <View className="h-2 overflow-hidden rounded-pill bg-white/20" accessibilityRole="progressbar">
      <View className="h-2 rounded-pill bg-white" style={{ width: `${Math.round(value * 100)}%` }} />
    </View>
  );
}

/** Pause/resume and finish. Finishing asks first; a mistaken tap would end the ride. */
export function TrackingControls({
  paused,
  busy,
  onPause,
  onResume,
  onFinish,
}: {
  paused: boolean;
  busy?: boolean;
  onPause: () => void;
  onResume: () => void;
  onFinish: () => void;
}) {
  const confirmFinish = () =>
    Alert.alert("Einheit beenden?", "Danach kannst du speichern.", [
      { text: "Weiter", style: "cancel" },
      { text: "Beenden", onPress: onFinish },
    ]);
  return (
    <View className="flex-row gap-3">
      {paused ? (
        <Button label="Fortsetzen" icon={Play} size="lg" className="flex-1" loading={busy} onPress={onResume} />
      ) : (
        <Button label="Pause" icon={Pause} size="lg" variant="secondary" className="flex-1" onPress={onPause} />
      )}
      <Button label="Beenden" icon={Square} size="lg" variant="outline" className="flex-1" disabled={busy} onPress={confirmFinish} />
    </View>
  );
}

/**
 * Stops the accidental exit of a running session: the Android back button asks first and
 * pauses before leaving (the session stays stored and can be continued from "Was heute?").
 */
export function useLeaveGuard(active: boolean, onPause: () => void) {
  const router = useRouter();
  useEffect(() => {
    if (!active || Platform.OS !== "android") return;
    const sub = BackHandler.addEventListener("hardwareBackPress", () => {
      Alert.alert("Einheit läuft", "Du kannst die Aufzeichnung pausieren und später fortsetzen.", [
        { text: "Weiter aufzeichnen", style: "cancel" },
        {
          text: "Pausieren und verlassen",
          onPress: () => {
            onPause();
            router.back();
          },
        },
      ]);
      return true;
    });
    return () => sub.remove();
  }, [active, onPause, router]);
}
