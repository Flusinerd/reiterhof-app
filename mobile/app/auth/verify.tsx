import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useRef, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { Brand } from "@/components/brand";
import { Button, PageHeader, Screen, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { isPlausibleToken } from "@/lib/deep-link";
import { colors } from "@/lib/theme";

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
      setError("Link ungültig. Fordere einen neuen an.");
      return;
    }
    signInWithMagicToken(token).catch((e) => setError(errorMessage(e)));
  }, [token, signInWithMagicToken]);

  return (
    <Screen contentClassName="pt-8">
      <Brand />
      <PageHeader
        title={error ? "Anmeldung fehlgeschlagen" : "Anmeldung läuft"}
        description={error ?? undefined}
        className="mt-4"
      />
      {error ? (
        <View className="gap-3">
          <Text variant="bodySm" tone="muted">
            Links gelten 15 Minuten und nur einmal.
          </Text>
          <Button label="Neu anfordern" size="lg" fullWidth onPress={() => router.replace("/(auth)/sign-in")} />
        </View>
      ) : (
        <ActivityIndicator color={colors.primary.DEFAULT} className="self-start" />
      )}
    </Screen>
  );
}
