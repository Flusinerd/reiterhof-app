import { useRouter } from "expo-router";
import { Download, FileText, Scale, Trash2 } from "lucide-react-native";
import { File, Paths } from "expo-file-system";
import * as Sharing from "expo-sharing";
import { useState } from "react";
import { Alert, Share, View } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { Button, Card, Divider, Hero, Icon, PressableCard, Screen, SectionLabel, Switch, Text } from "@/components/ui";
import { ApiError, errorMessage } from "@/lib/api";
import { privacyApi } from "@/lib/api/privacy";
import { useAuth } from "@/lib/auth";
import { useConsents, useSetConsent } from "@/lib/consent";
import {
  CONSENT_COPY,
  consentErrorMessage,
  deleteBlockedMessage,
  exportFileName,
  formatExport,
  orderedConsents,
  type ConsentKind,
} from "@/lib/consent-core";
import { disableGeofence } from "@/lib/geofence";

export default function PrivacySettings() {
  const router = useRouter();
  const { signOut } = useAuth();
  const consents = useConsents();
  const setConsent = useSetConsent();
  const prompt = useConsentPrompt();
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const items = orderedConsents(consents.data?.items);

  async function toggle(kind: ConsentKind, on: boolean) {
    setError(null);
    if (on) {
      // The sheet explains, and its "Erlauben" records the consent.
      await prompt.ensure(kind);
      return;
    }
    try {
      await setConsent.mutateAsync({ kind, granted: false });
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function exportData() {
    setError(null);
    setExporting(true);
    try {
      const data = await privacyApi.exportData();
      const text = formatExport(data);
      const file = new File(Paths.cache, exportFileName(new Date()));
      file.create({ overwrite: true });
      file.write(text);
      if (await Sharing.isAvailableAsync()) {
        await Sharing.shareAsync(file.uri, { mimeType: "application/json", dialogTitle: "Meine Reiterhof-Daten" });
      } else {
        await Share.share({ message: text });
      }
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setExporting(false);
    }
  }

  function confirmDelete() {
    Alert.alert(
      "Konto endgültig löschen?",
      "Dein Name, deine E-Mail-Adresse und deine Anwesenheiten werden gelöscht. Einträge, die für die Pferde wichtig bleiben, bleiben ohne deinen Namen erhalten. Das kann nicht rückgängig gemacht werden.",
      [
        { text: "Abbrechen", style: "cancel" },
        { text: "Konto löschen", style: "destructive", onPress: () => void deleteAccount() },
      ],
    );
  }

  async function deleteAccount() {
    setError(null);
    setDeleting(true);
    try {
      await privacyApi.deleteAccount();
      await disableGeofence();
      await signOut(); // the session is gone on the server; this drops the local one
    } catch (err) {
      setError((err instanceof ApiError ? deleteBlockedMessage(err.code) : null) ?? errorMessage(err));
      setDeleting(false);
    }
  }

  return (
    <Screen back>
      <Hero
        eyebrow="Einstellungen"
        title="Datenschutz"
        description="Hier entscheidest du, was die App von dir nutzen darf. Du kannst jede Erlaubnis jederzeit zurücknehmen."
      />

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      <SectionLabel>Erlaubnisse</SectionLabel>
      <Card className="gap-4">
        {consents.isPending ? (
          <Text variant="body" tone="muted">
            Wird geladen ...
          </Text>
        ) : consents.isError ? (
          <View className="gap-3">
            <Text variant="body" tone="danger">
              {(consents.error instanceof ApiError ? consentErrorMessage(consents.error.code) : null) ?? errorMessage(consents.error)}
            </Text>
            <Button label="Erneut versuchen" variant="outline" onPress={() => void consents.refetch()} />
          </View>
        ) : (
          items.map((item, i) => (
            <View key={item.kind} className="gap-2">
              {i > 0 ? <Divider className="mb-2" /> : null}
              <Switch
                label={CONSENT_COPY[item.kind].label}
                description={CONSENT_COPY[item.kind].summary}
                value={item.granted}
                disabled={setConsent.isPending}
                onValueChange={(on) => void toggle(item.kind, on)}
              />
              {item.granted && !item.up_to_date ? (
                <Text variant="caption">Der Datenschutztext hat sich geändert. Schalte kurz aus und wieder ein, um zuzustimmen.</Text>
              ) : null}
            </View>
          ))
        )}
      </Card>

      <SectionLabel>Deine Daten</SectionLabel>
      <Card className="gap-3">
        <Text variant="body">Lade alle Daten herunter, die die App über dich gespeichert hat. Du bekommst eine Datei im JSON-Format.</Text>
        <Button label="Meine Daten herunterladen" icon={Download} variant="outline" fullWidth loading={exporting} onPress={() => void exportData()} />
      </Card>

      <SectionLabel>Rechtliches</SectionLabel>
      <Card padded={false}>
        <PressableCard shape="tile" padded={false} className="min-h-touch flex-row items-center gap-3 border-0 px-5 py-3" onPress={() => router.push("/legal/privacy")}>
          <Icon as={FileText} size={20} className="text-muted" />
          <Text variant="body">Datenschutzerklärung</Text>
        </PressableCard>
        <Divider />
        <PressableCard shape="tile" padded={false} className="min-h-touch flex-row items-center gap-3 border-0 px-5 py-3" onPress={() => router.push("/legal/imprint")}>
          <Icon as={Scale} size={20} className="text-muted" />
          <Text variant="body">Impressum</Text>
        </PressableCard>
      </Card>

      <SectionLabel>Konto</SectionLabel>
      <Card className="gap-3">
        <Text variant="body">
          Beim Löschen werden dein Name, deine E-Mail-Adresse und deine Anwesenheiten entfernt. Besitzt du noch Pferde, übergib sie vorher an ein anderes Mitglied.
        </Text>
        <Button label="Konto löschen" icon={Trash2} variant="danger" fullWidth loading={deleting} onPress={confirmDelete} />
      </Card>
      {prompt.sheet}
    </Screen>
  );
}
