import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";

import { Button, Card, Hero, Screen, Text } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";

export default function CheckEmail() {
  const router = useRouter();
  const { email } = useLocalSearchParams<{ email?: string }>();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ text: string; error: boolean } | null>(null);

  async function resend() {
    if (!email) return;
    setBusy(true);
    setMessage(null);
    try {
      await api.requestMagicLink(email);
      setMessage({ text: "Wir haben dir einen neuen Link geschickt.", error: false });
    } catch (e) {
      setMessage({ text: errorMessage(e), error: true });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Screen back>
      <Hero
        eyebrow="Fast geschafft"
        title="Schau in dein Postfach"
        description={
          email
            ? `Wir haben einen Link an ${email} geschickt. Öffne ihn auf diesem Gerät. Er ist 15 Minuten gültig.`
            : "Wir haben dir einen Link geschickt. Öffne ihn auf diesem Gerät. Er ist 15 Minuten gültig."
        }
      />
      <Card className="gap-3">
        <Text variant="bodySm" tone="muted">
          Keine E-Mail bekommen? Schau auch im Spam-Ordner nach.
        </Text>
        <Button variant="secondary" label="Link erneut senden" fullWidth loading={busy} onPress={resend} />
        <Button variant="ghost" label="Andere Adresse verwenden" fullWidth onPress={() => router.back()} />
        {message ? (
          <Text variant="bodySm" tone={message.error ? "danger" : "primary"} accessibilityRole={message.error ? "alert" : undefined}>
            {message.text}
          </Text>
        ) : null}
      </Card>
    </Screen>
  );
}
