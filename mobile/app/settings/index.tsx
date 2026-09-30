import { useMutation, useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { BellOff, ChevronRight, LogOut, MapPin, Pencil, ShieldCheck, type LucideIcon } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, Pressable, View } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { InviteCard } from "@/components/invite-card";
import { ReminderTimeCard } from "@/components/reminder-time-card";
import { WebPushCard } from "@/components/web-push-card";
import {
  Button,
  Card,
  Divider,
  Icon,
  PageHeader,
  Screen,
  Section,
  Switch,
  Text,
} from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { reminderKeys, remindersApi, useNotificationSettings } from "@/lib/api/reminders";
import { useAuth } from "@/lib/auth";
import { useHasConsent } from "@/lib/consent";
import type { NotificationSettings } from "@/lib/reminders";
import { colors } from "@/lib/theme";

/**
 * Settings (JAN-70): notifications per kind with the stable's reminder time, inviting (admins only), then the
 * account (name, presence, privacy, sign-out).
 */
export default function Settings() {
  const queryClient = useQueryClient();
  const { user, hasStable, signOut } = useAuth();
  const settings = useNotificationSettings();
  const pushConsent = useHasConsent("push");
  const prompt = useConsentPrompt();
  const [error, setError] = useState<string | null>(null);
  const [signingOut, setSigningOut] = useState(false);

  const toggleKind = useMutation({
    mutationFn: ({ kind, enabled }: { kind: string; enabled: boolean }) => remindersApi.setNotification(kind, enabled),
    onMutate: async ({ kind, enabled }) => {
      setError(null);
      await queryClient.cancelQueries({ queryKey: reminderKeys.notifications });
      const previous = queryClient.getQueryData<NotificationSettings>(reminderKeys.notifications);
      // Switch at once; a failure restores the previous state.
      queryClient.setQueryData<NotificationSettings>(reminderKeys.notifications, (old) =>
        old ? { ...old, items: old.items.map((it) => (it.kind === kind ? { ...it, enabled } : it)) } : old,
      );
      return { previous };
    },
    onError: (e, _vars, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(reminderKeys.notifications, ctx.previous);
      setError(errorMessage(e));
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: reminderKeys.notifications }),
  });

  async function allowPush() {
    await prompt.ensure("push");
    void queryClient.invalidateQueries({ queryKey: reminderKeys.notifications });
  }

  async function leave() {
    setSigningOut(true);
    try {
      await signOut();
    } finally {
      setSigningOut(false);
    }
  }

  return (
    <Screen back>
      <PageHeader title="Einstellungen" description={user ? `${user.name}, ${user.email}` : undefined} />

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      <Section title="Benachrichtigungen">
        {pushConsent === false ? (
          <Card className="gap-3 border-accent-soft bg-accent-soft">
            <View className="flex-row items-center gap-3">
              <Icon as={BellOff} size={20} className="text-accent-text" />
              <Text variant="bodyStrong" className="flex-1">
                Mitteilungen sind aus
              </Text>
            </View>
            <Text variant="secondary">
              Ohne deine Erlaubnis kommen keine Mitteilungen, egal was du unten einstellst. Erinnerungen siehst du trotzdem in der Übersicht.
            </Text>
            <Button label="Erlauben" variant="outline" onPress={() => void allowPush()} />
          </Card>
        ) : null}
        <WebPushCard />
        <Card className="gap-4">
          {settings.isPending ? (
            <ActivityIndicator color={colors.primary.DEFAULT} />
          ) : settings.isError ? (
            <View className="gap-3">
              <Text variant="body" tone="danger">
                {errorMessage(settings.error)}
              </Text>
              <Button label="Erneut versuchen" variant="outline" onPress={() => void settings.refetch()} />
            </View>
          ) : (
            settings.data.items.map((item, i) => (
              <View key={item.kind}>
                {i > 0 ? <Divider className="mb-4" /> : null}
                <Switch
                  label={item.label}
                  description={item.description}
                  value={item.enabled}
                  onValueChange={(enabled) => toggleKind.mutate({ kind: item.kind, enabled })}
                />
              </View>
            ))
          )}
        </Card>
        <ReminderTimeCard />
      </Section>

      {user?.is_admin ? (
        <Section title="Stallgasse">
          <InviteCard />
        </Section>
      ) : null}

      <Section title="Konto">
        <Card padded={false}>
          <LinkRow
            icon={Pencil}
            title="Name"
            description={user?.name ?? "Ändern"}
            onPress={() => router.push("/(auth)/name" as Href)}
          />
          <Divider />
          {hasStable ? (
            <>
              <LinkRow
                icon={MapPin}
                title="Anwesenheit"
                description="Automatisch erkennen, Sichtbarkeit"
                onPress={() => router.push("/presence" as Href)}
              />
              <Divider />
            </>
          ) : null}
          <LinkRow
            icon={ShieldCheck}
            title="Datenschutz"
            description="Erlaubnisse, Datenexport, Konto löschen"
            onPress={() => router.push("/settings/privacy" as Href)}
          />
        </Card>
        <Button label="Abmelden" icon={LogOut} variant="outline" fullWidth loading={signingOut} onPress={() => void leave()} />
      </Section>
      {prompt.sheet}
    </Screen>
  );
}

/** A row in the account card that opens another screen. */
function LinkRow({ icon, title, description, onPress }: { icon: LucideIcon; title: string; description: string; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${title}. ${description}`}
      className="min-h-touch flex-row items-center gap-3 px-5 py-4"
      onPress={onPress}
    >
      <Icon as={icon} size={20} className="text-muted" />
      <View className="flex-1">
        <Text variant="bodyStrong">{title}</Text>
        <Text variant="secondary">{description}</Text>
      </View>
      <Icon as={ChevronRight} size={20} className="text-muted" />
    </Pressable>
  );
}
