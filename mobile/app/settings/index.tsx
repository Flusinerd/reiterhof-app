import { useMutation, useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { BellOff, ChevronRight, LogOut, Pencil, ShieldCheck } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, View } from "react-native";

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
  PressableCard,
  Screen,
  Section,
  Switch,
  Text,
  ToggleGroup,
  ToggleGroupItem,
} from "@/components/ui";
import { api, errorMessage, type PresenceVisibility } from "@/lib/api";
import { reminderKeys, remindersApi, useNotificationSettings } from "@/lib/api/reminders";
import { ME_KEY, useAuth } from "@/lib/auth";
import { useHasConsent } from "@/lib/consent";
import { VISIBILITY_OPTIONS, visibilityDescription } from "@/lib/presence-format";
import type { NotificationSettings } from "@/lib/reminders";
import { colors } from "@/lib/theme";

/** Settings (JAN-70): notifications, inviting (admins), the stable's reminder time, presence, privacy, account. */
export default function Settings() {
  const queryClient = useQueryClient();
  const { user, signOut } = useAuth();
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

  const changeVisibility = useMutation({
    mutationFn: (visibility: PresenceVisibility) => api.updateMe({ presence_visibility: visibility }),
    onMutate: () => setError(null),
    onSuccess: (updated) => queryClient.setQueryData(ME_KEY, updated),
    onError: (e) => setError(errorMessage(e)),
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

  const visibility = (user?.presence_visibility ?? "all") as PresenceVisibility;

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
      </Section>

      <Section title="Stallgasse">
        {user?.is_admin ? <InviteCard /> : null}
        <ReminderTimeCard />
      </Section>

      <Section title="Anwesenheit" description="Wer sieht mich?">
        <ToggleGroup type="single" value={visibility} onValueChange={(v) => changeVisibility.mutate(v as PresenceVisibility)}>
          {VISIBILITY_OPTIONS.map((o) => (
            <ToggleGroupItem key={o.value} value={o.value} label={o.label} disabled={changeVisibility.isPending} />
          ))}
        </ToggleGroup>
        <Text variant="secondary">{visibilityDescription(visibility)}</Text>
      </Section>

      <Section title="Datenschutz">
        <PressableCard
          padded={false}
          className="min-h-touch flex-row items-center gap-3 px-5 py-4"
          onPress={() => router.push("/settings/privacy" as Href)}
        >
          <Icon as={ShieldCheck} size={20} className="text-muted" />
          <View className="flex-1">
            <Text variant="bodyStrong">Datenschutz</Text>
            <Text variant="secondary">Erlaubnisse, Datenexport, Konto löschen</Text>
          </View>
          <Icon as={ChevronRight} size={20} className="text-muted" />
        </PressableCard>
      </Section>

      <Section title="Konto">
        <Button
          label="Name ändern"
          icon={Pencil}
          variant="outline"
          fullWidth
          onPress={() => router.push("/(auth)/name" as Href)}
        />
        <Button label="Abmelden" icon={LogOut} variant="outline" fullWidth loading={signingOut} onPress={() => void leave()} />
      </Section>
      {prompt.sheet}
    </Screen>
  );
}
