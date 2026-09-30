import { useRouter } from "expo-router";
import { useState } from "react";
import { View } from "react-native";

import { LegalLinks } from "@/components/consent-legal-links";
import { Button, Input, PageHeader, Screen, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { formatInviteCode, normalizeInviteCode } from "@/lib/validation";

export default function Join() {
  const router = useRouter();
  const { joinStable, signOut, user } = useAuth();
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function join() {
    if (normalizeInviteCode(code).length < 8) {
      setError("Der Code hat 8 Zeichen, z. B. ABCD-EFGH.");
      return;
    }
    setError(null);
    setBusy(true);
    try {
      // The root layout moves on to the tabs as soon as the profile has a stable.
      await joinStable(normalizeInviteCode(code));
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  }

  return (
    <Screen keyboardShouldPersistTaps="handled" contentClassName="flex-grow pt-8">
      <PageHeader
        eyebrow={user ? `Hallo ${user.name}` : undefined}
        title="Stall beitreten"
        description="Gib den Einladungscode deines Stalls ein."
      />

      <View className="gap-3">
        <Input
          value={code}
          onChangeText={(v) => setCode(formatInviteCode(v))}
          placeholder="ABCD-EFGH"
          accessibilityLabel="Einladungscode"
          autoCapitalize="characters"
          autoCorrect={false}
          returnKeyType="go"
          onSubmitEditing={join}
          className="h-16 text-center font-display text-title"
          style={{ letterSpacing: 3 }}
        />
        <Button label="Beitreten" size="lg" fullWidth loading={busy} onPress={join} />
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
      </View>

      <View className="mt-auto gap-2">
        <Button variant="ghost" label="Datenschutz und Konto" fullWidth onPress={() => router.push("/settings/privacy")} />
        <Button variant="ghost" label="Abmelden" fullWidth onPress={signOut} />
        <LegalLinks />
      </View>
    </Screen>
  );
}
