import { Stack, useLocalSearchParams, useRouter } from "expo-router";
import { Play } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { TrackingMap } from "@/components/tracking-map";
import { GaitChip, GaitLegend, StatTile, TrackingControls, useLeaveGuard } from "@/components/tracking-parts";
import { Button, Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";
import { formatClock, isActivity, activityLabel } from "@/lib/training";
import { formatDistance, formatElevation, formatSpeed } from "@/lib/tracking-format";
import { startFailureText, type StartResult } from "@/lib/tracking-location";
import { useStoredSession, useTrackingSession } from "@/lib/tracking-session";
import type { Snapshot } from "@/lib/tracking-persist";

type Params = { horse?: string; activity?: string; minutes?: string; exercise?: string; resume?: string };

/** Ride tracking with GPS (JAN-63): track, distance, speed, elevation, map colored by gait. */
export default function GpsTrack() {
  const params = useLocalSearchParams<Params>();
  const stored = useStoredSession(params.resume === "1");
  if (stored.loading) {
    return (
      <Screen scroll={false} className="justify-center">
        <ActivityIndicator accessibilityLabel="Einheit wird geladen" />
      </Screen>
    );
  }
  const snapshot = params.resume === "1" ? stored.snapshot : null;
  const horse = snapshot?.horse ?? params.horse;
  const activity = snapshot?.activity ?? params.activity;
  if (!horse || !isActivity(activity)) {
    return (
      <Screen back>
        <Text variant="body">Für diese Einheit fehlen Angaben. Bitte starte sie im Tab „Training“.</Text>
      </Screen>
    );
  }
  return <GpsRun horse={horse} activity={activity} exerciseId={snapshot?.exerciseId ?? params.exercise} snapshot={snapshot} />;
}

function GpsRun({
  horse,
  activity,
  exerciseId,
  snapshot,
}: {
  horse: string;
  activity: string;
  exerciseId?: string;
  snapshot: Snapshot | null;
}) {
  const router = useRouter();
  const t = useTrackingSession({ mode: "gps", horse, activity, exerciseId, initial: snapshot });
  const [started, setStarted] = useState(snapshot !== null);
  const [failure, setFailure] = useState<Exclude<StartResult, { ok: true }>["reason"] | null>(null);
  const [noBackground, setNoBackground] = useState(false);
  const [busy, setBusy] = useState(false);
  useLeaveGuard(started, t.pause);

  const start = async () => {
    setBusy(true);
    const result = await t.begin();
    setBusy(false);
    if (result.ok) {
      setFailure(null);
      setNoBackground(!result.background);
      setStarted(true);
    } else {
      setFailure(result.reason);
    }
  };

  const resume = async () => {
    setBusy(true);
    const result = await t.resume();
    setBusy(false);
    if (result && !result.ok) setFailure(result.reason);
    else {
      setFailure(null);
      if (result?.ok) setNoBackground(!result.background);
    }
  };

  const finish = async () => {
    setBusy(true);
    const { params } = await t.finish();
    router.replace({ pathname: "/training/session/finish", params });
  };

  const seconds = t.stats.activeSeconds;
  const label = activityLabel(activity);

  if (!started) {
    return (
      <Screen back>
        <Hero tone="deep" eyebrow={label} title="Bereit für den Ausritt" description="Route, Strecke, Tempo und Höhenmeter werden aufgezeichnet." />
        <Card className="gap-2">
          <Text variant="bodyStrong">Ortung und Gangarten</Text>
          <Text variant="secondary">
            Reiterhof nutzt dafür deinen Standort, auch wenn der Bildschirm aus ist. Die Gangart erkennt das Telefon an
            der Bewegung; trage es am besten in der Jackentasche oder am Körper.
          </Text>
        </Card>
        {failure ? (
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {startFailureText(failure)}
          </Text>
        ) : null}
        <Button label="Aufzeichnung starten" icon={Play} size="lg" fullWidth loading={busy} onPress={start} />
      </Screen>
    );
  }

  return (
    <Screen>
      <Stack.Screen options={{ gestureEnabled: false }} />
      <Hero tone="deep" eyebrow={`${label}${t.paused ? " · pausiert" : ""}`} value={formatClock(seconds)} valueSize="lg">
        {t.paused ? (
          <Text variant="secondary" className="text-white/70">
            Die Aufzeichnung ist angehalten.
          </Text>
        ) : (
          <GaitChip gait={t.gait ?? (t.points.length > 0 ? t.points[t.points.length - 1]!.gait : null)} />
        )}
      </Hero>

      <TrackingMap points={t.points} follow={!t.paused} />
      <GaitLegend />

      <View className="flex-row gap-3">
        <StatTile label="Strecke" value={formatDistance(t.stats.distanceM)} />
        <StatTile label="Ø Tempo" value={formatSpeed(t.stats.avgSpeedKmh)} />
        <StatTile label="Aufstieg" value={formatElevation(t.stats.elevationGainM)} />
      </View>

      {t.points.length === 0 && !t.paused ? (
        <Text variant="secondary">Warte auf das GPS-Signal. Draußen mit freier Sicht geht es am schnellsten.</Text>
      ) : null}
      {noBackground ? (
        <Text variant="secondary" tone="accent">
          Ohne die Erlaubnis „Immer“ für den Standort wird nur aufgezeichnet, solange die App geöffnet bleibt.
        </Text>
      ) : null}
      {failure ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {startFailureText(failure)}
        </Text>
      ) : null}
      {t.sensorAvailable === false ? (
        <Text variant="secondary">Kein Bewegungssensor gefunden: Die Gangart wird aus dem GPS-Tempo geschätzt.</Text>
      ) : null}

      <SectionLabel>Aufzeichnung</SectionLabel>
      <TrackingControls paused={t.paused} busy={busy} onPause={t.pause} onResume={resume} onFinish={finish} />
    </Screen>
  );
}
