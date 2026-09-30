import { useRouter } from "expo-router";
import { useState } from "react";
import { Platform } from "react-native";

import { LegalLinks } from "@/components/consent-legal-links";
import { AppleSignInButton, GoogleSignInButton } from "@/components/social-sign-in";
import { Button, Card, Hero, Input, Screen, SectionLabel, Text } from "@/components/ui";
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
      setError("Bitte gib eine gültige E-Mail-Adresse ein.");
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
    <Screen keyboardShouldPersistTaps="handled">
      <Hero
        eyebrow="Stallfunk"
        title="Willkommen"
        description="Melde dich ohne Passwort an. Wir schicken dir einen Link per E-Mail."
      />

      <SectionLabel>Mit E-Mail</SectionLabel>
      <Card className="gap-3">
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
        />
        <Button label="Link senden" fullWidth loading={busy} onPress={sendLink} />
      </Card>

      <SectionLabel>Oder</SectionLabel>
      <Card className="gap-3">
        <GoogleSignInButton onError={setError} />
        {Platform.OS === "ios" ? <AppleSignInButton onError={setError} /> : null}
      </Card>

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      <LegalLinks />
    </Screen>
  );
}
