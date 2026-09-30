import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { View } from "react-native";

import { LegalLinks } from "@/components/consent-legal-links";
import { Button, Divider, Input, PageHeader, Screen, Text } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { ME_KEY, useAuth } from "@/lib/auth";
import { cooldownRemainingSeconds, RESEND_COOLDOWN_SECONDS, resendLabel } from "@/lib/login-code";
import { normalizeEmail } from "@/lib/validation";

/**
 * Age confirmation (Art. 8 GDPR, JAN-86). Location, photos, maps and push rest on consent,
 * which a person under 16 cannot give alone. So after the name every account states once:
 * 16 or older (one tap), or younger, in which case a parent gets a mail with a confirmation
 * link. Until the parent confirmed, the root layout keeps sending the user here; the
 * privacy settings (export, delete) stay reachable.
 */
export default function Age() {
  const router = useRouter();
  const queryClient = useQueryClient();
  // Once confirmed (by the person or the parent) the root layout moves on to the stable or the tabs.
  const { user, refresh, signOut } = useAuth();
  const pending = user?.age_status === "parent_pending";
  const [mode, setMode] = useState<"choose" | "parent">("choose");
  const [parentEmail, setParentEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [availableAt, setAvailableAt] = useState(0);
  const [secondsLeft, setSecondsLeft] = useState(0);

  useEffect(() => {
    const tick = () => setSecondsLeft(cooldownRemainingSeconds(availableAt, Date.now()));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [availableAt]);

  // While the parent has not answered, look for the confirmation now and then.
  useEffect(() => {
    if (!pending) return;
    const id = setInterval(() => void refresh(), 15_000);
    return () => clearInterval(id);
  }, [pending, refresh]);

  async function over16() {
    setError(null);
    setBusy(true);
    try {
      queryClient.setQueryData(ME_KEY, await api.confirmAge());
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  }

  async function sendToParent(address?: string) {
    const normalized = normalizeEmail(address ?? parentEmail);
    if (!normalized) {
      setError("Ungültige E-Mail-Adresse.");
      return;
    }
    if (user && normalized === user.email.toLowerCase()) {
      setError("Bitte die Adresse eines Elternteils, nicht deine eigene.");
      return;
    }
    setError(null);
    setBusy(true);
    try {
      await api.requestParentalConsent(normalized);
      setAvailableAt(Date.now() + RESEND_COOLDOWN_SECONDS * 1000);
      await refresh();
      setMode("choose");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  if (pending && mode === "choose") {
    return (
      <Screen keyboardShouldPersistTaps="handled" contentClassName="flex-grow pt-8">
        <PageHeader
          title="Warten auf deine Eltern"
          description={`Wir haben eine E-Mail an ${user?.parent_email ?? "deine Eltern"} geschickt. Sobald dort zugestimmt wurde, geht es hier weiter.`}
        />
        <View className="gap-3">
          <Text variant="bodySm" tone="muted">
            Der Link in der Mail gilt 7 Tage. Nichts angekommen? Spam-Ordner prüfen oder die Mail erneut senden.
          </Text>
          <Button label="Status prüfen" size="lg" fullWidth loading={busy} onPress={() => void refresh()} />
          <Button
            variant="secondary"
            label={resendLabel(secondsLeft)}
            fullWidth
            disabled={secondsLeft > 0 || busy}
            onPress={() => void sendToParent(user?.parent_email ?? undefined)}
          />
          <Button variant="ghost" label="Andere Adresse" fullWidth disabled={busy} onPress={() => setMode("parent")} />
          {error ? (
            <Text variant="bodySm" tone="danger" accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
        </View>
        <Divider />
        <View className="gap-2">
          <Button variant="ghost" label="Ich bin doch 16 oder älter" fullWidth disabled={busy} onPress={() => void over16()} />
          <Button variant="ghost" label="Datenschutz und Konto" fullWidth onPress={() => router.push("/settings/privacy")} />
          <Button variant="ghost" label="Abmelden" fullWidth onPress={signOut} />
        </View>
        <LegalLinks className="mt-auto" />
      </Screen>
    );
  }

  if (mode === "parent") {
    return (
      <Screen keyboardShouldPersistTaps="handled" contentClassName="flex-grow pt-8">
        <PageHeader
          title="E-Mail an deine Eltern"
          description="Ein Elternteil bekommt eine Mail mit dem, was die App speichert, und stimmt mit einem Klick zu."
        />
        <View className="gap-3">
          <Input
            value={parentEmail}
            onChangeText={setParentEmail}
            placeholder="E-Mail-Adresse eines Elternteils"
            accessibilityLabel="E-Mail-Adresse eines Elternteils"
            keyboardType="email-address"
            autoCapitalize="none"
            autoComplete="email"
            autoCorrect={false}
            textContentType="emailAddress"
            returnKeyType="send"
            onSubmitEditing={() => void sendToParent()}
            className="h-14"
          />
          <Button label="Mail senden" size="lg" fullWidth loading={busy} onPress={() => void sendToParent()} />
          <Button variant="ghost" label="Zurück" fullWidth disabled={busy} onPress={() => setMode("choose")} />
          {error ? (
            <Text variant="bodySm" tone="danger" accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
        </View>
        <LegalLinks className="mt-auto" />
      </Screen>
    );
  }

  return (
    <Screen contentClassName="flex-grow pt-8">
      <PageHeader
        title="Wie alt bist du?"
        description="Standort, Fotos und Mitteilungen brauchen deine Einwilligung. Unter 16 muss ein Elternteil zustimmen."
      />
      <View className="gap-3">
        <Button label="Ich bin 16 oder älter" size="lg" fullWidth loading={busy} onPress={() => void over16()} />
        <Button label="Ich bin jünger als 16" size="lg" variant="outline" fullWidth disabled={busy} onPress={() => setMode("parent")} />
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        <Text variant="bodySm" tone="muted">
          Wir speichern kein Geburtsdatum, nur deine Angabe. Deine Daten kannst du jederzeit unter Datenschutz einsehen und löschen.
        </Text>
      </View>
      <View className="mt-auto gap-2">
        <Button variant="ghost" label="Abmelden" fullWidth onPress={signOut} />
        <LegalLinks />
      </View>
    </Screen>
  );
}
