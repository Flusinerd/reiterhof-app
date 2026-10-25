import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { BookOpen, MapPin, Warehouse } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import { Button, Icon, PageHeader, Screen, Section, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { activityLabel, isActivity, type Activity } from "@/lib/training";
import { snapshotSummary } from "@/lib/tracking-persist";
import { useStoredSession } from "@/lib/tracking-session";
import { clearSnapshot } from "@/lib/tracking-store";
import { stopTracking } from "@/lib/tracking-location";

type Params = { horse?: string; activity?: string; minutes?: string; exercise?: string; focus?: string };

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
        ...(params.focus ? { focus: params.focus } : {}),
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
    <View className="gap-4 rounded-card border border-border bg-raised p-5">
      <View className="gap-1">
        <Text variant="eyebrow" tone="brass">
          Noch offen
        </Text>
        <Text variant="titleLg">Es läuft noch eine Einheit</Text>
        <Text variant="bodySm" tone="muted">
          {snapshotSummary(running, activityLabel(running.activity))}
        </Text>
      </View>
      <View className="gap-2">
        <Button label="Fortsetzen" fullWidth onPress={resume} />
        <Button label="Verwerfen" variant="ghost" size="sm" fullWidth onPress={() => void discard()} />
      </View>
    </View>
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
      <PageHeader
        eyebrow="Einheit starten"
        title={activityLabel(activity)}
        description={params.minutes ? `Empfohlen: ${params.minutes} Minuten.` : undefined}
      />

      {resumeCard}

      <Section title="Aufzeichnung">
        <RowList>
          {modes.map((m, i) => (
            <ListRow
              key={m.mode}
              leading={
                <View className="h-11 w-11 items-center justify-center rounded-pill bg-primary-soft">
                  <Icon as={m.icon} size={22} className="text-primary-deep" />
                </View>
              }
              eyebrow={i === 0 ? "Empfohlen" : undefined}
              eyebrowTone="brass"
              title={m.title}
              detail={m.text}
              accessibilityLabel={`${m.title}${i === 0 ? ", empfohlen" : ""}`}
              onPress={() => go(m.mode)}
            />
          ))}
        </RowList>
      </Section>

      <ListRow
        leading={<Icon as={BookOpen} size={20} className="text-muted" />}
        title="Übungen"
        accessibilityLabel="Übungen öffnen"
        onPress={() => router.push("/training/exercises" as Href)}
      />
    </Screen>
  );
}
