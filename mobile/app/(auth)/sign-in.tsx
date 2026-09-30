import { useRouter } from "expo-router";
import { useState } from "react";
import { Platform, View } from "react-native";

import { Brand } from "@/components/brand";
import { LegalLinks } from "@/components/consent-legal-links";
import { AppleSignInButton, GoogleSignInButton } from "@/components/social-sign-in";
import { Button, Divider, Input, PageHeader, Screen, Text } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { normalizeEmail } from "@/lib/validation";

export default function SignIn() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function sendLink() {
    const normalized = normalizeEmail(email);
    if (!normalized) {
      setError("Ungültige E-Mail-Adresse.");
      return;
    }
    setError(null);
    setBusy(true);
    try {
      await api.requestMagicLink(normalized);
      router.push({ pathname: "/check-email", params: { email: normalized } });
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Screen keyboardShouldPersistTaps="handled" contentClassName="flex-grow pt-8">
      <Brand />

      <PageHeader title="Anmelden" description="Ohne Passwort, per Code aus der E-Mail." className="mt-4" />

      <View className="gap-3">
        <Input
          value={email}
          onChangeText={setEmail}
          placeholder="E-Mail-Adresse"
          accessibilityLabel="E-Mail-Adresse"
          keyboardType="email-address"
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect={false}
          textContentType="emailAddress"
          returnKeyType="send"
          onSubmitEditing={sendLink}
          className="h-14"
        />
        <Button label="Code senden" size="lg" fullWidth loading={busy} onPress={sendLink} />
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
      </View>

      <Divider label="oder" />

      <View className="gap-3">
        <GoogleSignInButton onError={setError} />
        {Platform.OS === "ios" ? <AppleSignInButton onError={setError} /> : null}
      </View>

      <LegalLinks className="mt-auto" />
    </Screen>
  );
}
