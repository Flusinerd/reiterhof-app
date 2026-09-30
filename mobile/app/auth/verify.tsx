import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useRef, useState } from "react";
import { ActivityIndicator } from "react-native";

import { Button, Card, Hero, Screen, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { isPlausibleToken } from "@/lib/deep-link";

/**
 * Target of the magic link `stallfunk://auth/verify?token=...` (Expo Router maps the link to
 * this route). Exchanges the token for a session; the root layout then moves on to the
 * join screen or the tabs.
 */
export default function Verify() {
  const router = useRouter();
  const { token } = useLocalSearchParams<{ token?: string }>();
  const { signInWithMagicToken } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return; // a token works once; never send it twice
    started.current = true;
    if (!isPlausibleToken(token)) {
      setError("Der Link ist ungültig. Fordere einen neuen an.");
      return;
    }
    signInWithMagicToken(token).catch((e) => setError(errorMessage(e)));
  }, [token, signInWithMagicToken]);

  return (
    <Screen>
      <Hero
        eyebrow="Stallfunk"
        title={error ? "Anmeldung fehlgeschlagen" : "Du wirst angemeldet"}
        description={error ?? "Einen Moment bitte."}
        tone={error ? "warm" : "forest"}
      />
      {error ? (
        <Card className="gap-3">
          <Text variant="bodySm" tone="muted">
            Links gelten 15 Minuten und nur einmal.
          </Text>
          <Button label="Neuen Link anfordern" fullWidth onPress={() => router.replace("/(auth)/sign-in")} />
        </Card>
      ) : (
        <ActivityIndicator />
      )}
    </Screen>
  );
}
