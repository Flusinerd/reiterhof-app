import { Bell } from "lucide-react-native";
import { useState } from "react";
import { Platform, View } from "react-native";

import { Button, Card, Icon, Text } from "@/components/ui";
import { useHasConsent } from "@/lib/consent";
import { enableWebPush, webPushPermission, webPushSupportNow } from "@/lib/webpush";
import { webPushCardState, webPushReasonMessage, type WebPushReason } from "@/lib/webpush-core";

/**
 * Web only (JAN-74): the push consent covers all devices, but a browser needs its own permission
 * and subscription. Shows a hint when this browser cannot receive pushes (on iOS: "Zum
 * Home-Bildschirm hinzufügen, dann Mitteilungen erlauben") or a button that asks for the permission
 * from a tap. Renders nothing on native, without the push consent, or once everything is on.
 */
export function WebPushCard() {
  const consent = useHasConsent("push");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<WebPushReason | null>(null);
  const [, refresh] = useState(0);

  const state = webPushCardState({
    isWeb: Platform.OS === "web",
    consent,
    support: webPushSupportNow(),
    permission: webPushPermission(),
  });
  if (state.kind === "hidden") return null;

  async function enable() {
    setBusy(true);
    setFailure(null);
    const result = await enableWebPush();
    setBusy(false);
    if (!result.ok) setFailure(result.reason);
    refresh((n) => n + 1);
  }

  const reason = state.kind === "hint" ? state.reason : failure;
  return (
    <Card className="gap-3">
      <View className="flex-row items-center gap-3">
        <Icon as={Bell} size={20} className="text-accent-text" />
        <Text variant="bodyStrong" className="flex-1">
          Mitteilungen auf diesem Gerät
        </Text>
      </View>
      {reason ? (
        <Text variant="secondary" accessibilityRole={failure ? "alert" : undefined}>
          {webPushReasonMessage(reason)}
        </Text>
      ) : (
        <Text variant="secondary">Erinnerungen und Alarme brauchen die Erlaubnis dieses Browsers.</Text>
      )}
      {state.kind === "enable" ? (
        <Button label="Aktivieren" variant="outline" loading={busy} onPress={() => void enable()} />
      ) : null}
    </Card>
  );
}
