import { useLocalSearchParams, useRouter } from "expo-router";
import { Square } from "lucide-react-native";
import { useEffect, useRef, useState } from "react";

import { Button, Hero, Screen, Text } from "@/components/ui";
import { activityLabel, formatClock, isActivity, secondsToMinutes } from "@/lib/training";

/**
 * Placeholder of the tracking screen: a plain timer. The tracked session with gaits and rein
 * changes replaces it later and hands `gait` / `rein` (JSON) to the finish screen the same way.
 */
export default function TrainingSession() {
  const router = useRouter();
  const params = useLocalSearchParams<{ horse?: string; activity?: string; minutes?: string; exercise?: string }>();
  const activity = isActivity(params.activity) ? params.activity : null;
  const startedAt = useRef(new Date());
  const [seconds, setSeconds] = useState(0);

  useEffect(() => {
    const id = setInterval(() => setSeconds(Math.floor((Date.now() - startedAt.current.getTime()) / 1000)), 1000);
    return () => clearInterval(id);
  }, []);

  if (!params.horse || !activity) {
    return (
      <Screen back>
        <Text variant="body">Für diese Einheit fehlen Angaben. Bitte starte sie im Tab „Training“.</Text>
      </Screen>
    );
  }

  const finish = () => {
    router.replace({
      pathname: "/training/session/finish",
      params: {
        horse: params.horse!,
        activity,
        minutes: String(secondsToMinutes(seconds)),
        started_at: startedAt.current.toISOString(),
        ...(params.exercise ? { exercise: params.exercise } : {}),
      },
    });
  };

  return (
    <Screen back>
      <Hero eyebrow={activityLabel(activity)} value={formatClock(seconds)} valueSize="lg" tone="deep">
        <Text variant="secondary" className="text-white/70">
          {params.minutes ? `Empfohlen: ${params.minutes} Min.` : "Die Zeit läuft."}
        </Text>
      </Hero>
      <Text variant="secondary">Gangarten und Handwechsel werden in einer späteren Version automatisch erfasst.</Text>
      <Button label="Beenden" icon={Square} size="lg" fullWidth onPress={finish} />
    </Screen>
  );
}
