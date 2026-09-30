import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { BookOpen, ChevronRight, MapPin, Warehouse } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import { Button, Card, Hero, Icon, PressableCard, Screen, SectionLabel, Text } from "@/components/ui";
import { activityLabel, isActivity, type Activity } from "@/lib/training";
import { snapshotSummary } from "@/lib/tracking-persist";
import { useStoredSession } from "@/lib/tracking-session";
import { clearSnapshot } from "@/lib/tracking-store";
import { stopTracking } from "@/lib/tracking-location";

type Params = { horse?: string; activity?: string; minutes?: string; exercise?: string };

/** Ausritte are tracked with GPS; hall, arena and lunge work without it (JAN-63, JAN-65). */
const DEFAULT_MODE: Record<Activity, "gps" | "indoor"> = {
  hack: "gps",
  hall: "indoor",
  arena: "indoor",
  lunge: "indoor",
  jumping: "indoor",
  groundwork: "indoor",
  walker: "indoor",
};

/**
 * Start of a session ("Starten" in "Was heute?"): pick the tracking mode, or continue a
 * session that was still running when the app was closed.
 */
export default function TrainingSession() {
  const router = useRouter();
  const params = useLocalSearchParams<Params>();
  const activity = isActivity(params.activity) ? params.activity : null;
  const stored = useStoredSession(true, false);
  const [dropped, setDropped] = useState(false);
  const running = dropped ? null : stored.snapshot;

  const go = (mode: "gps" | "indoor") => {
    router.replace({
      pathname: mode === "gps" ? "/training/track/gps" : "/training/track/indoor",
      params: {
        horse: params.horse!,
        activity: activity!,
        ...(params.minutes ? { minutes: params.minutes } : {}),
        ...(params.exercise ? { exercise: params.exercise } : {}),
      },
    });
  };

  const resume = () => {
    if (!running) return;
    router.replace({
      pathname: running.mode === "gps" ? "/training/track/gps" : "/training/track/indoor",
      params: { resume: "1" },
    });
  };

  const discard = async () => {
    await stopTracking();
    await clearSnapshot();
    setDropped(true);
  };

  const resumeCard = running ? (
    <>
      <SectionLabel>Noch offen</SectionLabel>
      <Card className="gap-3">
        <Text variant="bodyStrong">Es läuft noch eine Einheit</Text>
        <Text variant="secondary">{snapshotSummary(running, activityLabel(running.activity))}</Text>
        <View className="flex-row gap-3">
          <Button label="Fortsetzen" className="flex-1" onPress={resume} />
          <Button label="Verwerfen" variant="outline" className="flex-1" onPress={() => void discard()} />
        </View>
      </Card>
    </>
  ) : null;

  if (!params.horse || !activity) {
    return (
      <Screen back>
        <Text variant="body">Angaben fehlen. Starte die Einheit im Tab „Training“.</Text>
        {resumeCard}
      </Screen>
    );
  }

  const preferred = DEFAULT_MODE[activity];
  const modes = [
    {
      mode: "gps" as const,
      icon: MapPin,
      title: "Mit GPS",
      text: "Strecke, Tempo, Höhe und Karte. Für Ausritte.",
    },
    {
      mode: "indoor" as const,
      icon: Warehouse,
      title: "Drinnen, ohne GPS",
      text: "Gangart per Sensor, Handwechsel, Übungsablauf. Für Halle, Platz, Longe.",
    },
  ].sort((a, b) => (a.mode === preferred ? -1 : b.mode === preferred ? 1 : 0));

  return (
    <Screen back>
      <Hero
        tone="deep"
        eyebrow="Einheit starten"
        title={activityLabel(activity)}
        description={params.minutes ? `Empfohlen: ${params.minutes} Minuten.` : undefined}
      />

      {resumeCard}

      <SectionLabel>Aufzeichnung</SectionLabel>
      <View className="gap-3">
        {modes.map((m, i) => (
          <PressableCard
            key={m.mode}
            shape="tile"
            className="flex-row items-center gap-4"
            accessibilityLabel={`${m.title}${i === 0 ? ", empfohlen" : ""}`}
            onPress={() => go(m.mode)}
          >
            <View className="h-11 w-11 items-center justify-center rounded-pill bg-primary-soft">
              <Icon as={m.icon} size={22} className="text-primary-deep" />
            </View>
            <View className="flex-1 gap-1">
              <Text variant="bodyStrong">{i === 0 ? `${m.title} (empfohlen)` : m.title}</Text>
              <Text variant="secondary">{m.text}</Text>
            </View>
            <Icon as={ChevronRight} size={20} className="text-muted" />
          </PressableCard>
        ))}
      </View>

      <PressableCard
        shape="tile"
        className="flex-row items-center gap-3"
        accessibilityLabel="Übungen öffnen"
        onPress={() => router.push("/training/exercises" as Href)}
      >
        <Icon as={BookOpen} size={20} className="text-primary-deep" />
        <Text variant="bodyStrong" className="flex-1">
          Übungen
        </Text>
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </PressableCard>
    </Screen>
  );
}
