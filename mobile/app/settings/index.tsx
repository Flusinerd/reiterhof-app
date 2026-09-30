import { useMutation, useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { BellOff, ChevronRight, LogOut, ShieldCheck } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { ReminderTimeCard } from "@/components/reminder-time-card";
import {
  Button,
  Card,
  Divider,
  Hero,
  Icon,
  PressableCard,
  Screen,
  SectionLabel,
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

/** Settings (JAN-70): notifications per kind, the stable's reminder time, presence, privacy, account. */
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
      <Hero
        eyebrow="Reiterhof"
        title="Einstellungen"
        description="Was dich erinnert, wann der Stall ans Decken denkt und wer dich im Stall sieht."
      />

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      <SectionLabel>Benachrichtigungen</SectionLabel>
      {pushConsent === false ? (
        <Card className="gap-3">
          <View className="flex-row items-center gap-3">
            <Icon as={BellOff} size={20} className="text-accent-text" />
            <Text variant="bodyStrong" className="flex-1">
              Mitteilungen sind ausgeschaltet
            </Text>
          </View>
          <Text variant="secondary">
            Ohne deine Erlaubnis schickt die App keine Mitteilungen, egal was du unten einstellst. Die Erinnerungen findest du trotzdem in der Übersicht.
          </Text>
          <Button label="Mitteilungen erlauben" variant="outline" onPress={() => void allowPush()} />
        </Card>
      ) : null}
      <Card className="gap-4">
        {settings.isPending ? (
          <ActivityIndicator />
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

      <SectionLabel>Stallgasse</SectionLabel>
      <ReminderTimeCard />

      <SectionLabel>Anwesenheit</SectionLabel>
      <Card className="gap-2">
        <Text variant="bodyStrong">Wer sieht mich?</Text>
        <ToggleGroup type="single" value={visibility} onValueChange={(v) => changeVisibility.mutate(v as PresenceVisibility)}>
          {VISIBILITY_OPTIONS.map((o) => (
            <ToggleGroupItem key={o.value} value={o.value} label={o.label} disabled={changeVisibility.isPending} />
          ))}
        </ToggleGroup>
        <Text variant="secondary">{visibilityDescription(visibility)}</Text>
      </Card>

      <SectionLabel>Datenschutz</SectionLabel>
      <PressableCard
        padded={false}
        className="min-h-touch flex-row items-center gap-3 px-5 py-4"
        onPress={() => router.push("/settings/privacy" as Href)}
      >
        <Icon as={ShieldCheck} size={20} className="text-muted" />
        <View className="flex-1">
          <Text variant="bodyStrong">Datenschutz und Einwilligungen</Text>
          <Text variant="secondary">Erlaubnisse, Datenexport und Konto löschen</Text>
        </View>
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </PressableCard>

      <SectionLabel>Konto</SectionLabel>
      <Card className="gap-3">
        <View>
          <Text variant="bodyStrong">{user?.name ?? ""}</Text>
          <Text variant="secondary">{user?.email ?? ""}</Text>
        </View>
        <Button label="Abmelden" icon={LogOut} variant="outline" fullWidth loading={signingOut} onPress={() => void leave()} />
      </Card>
      {prompt.sheet}
    </Screen>
  );
}
