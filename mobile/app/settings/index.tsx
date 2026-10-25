import { useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { Activity, Bell, BellOff, BookOpen, LogOut, MapPin, Pencil, ShieldCheck, type LucideIcon } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { Horse } from "@/components/icons/horse";
import { InviteCard } from "@/components/invite-card";
import { ReminderTimeCard } from "@/components/reminder-time-card";
import { WebPushCard } from "@/components/web-push-card";
import { Button, Icon, PageHeader, Screen, Section, Switch, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { errorMessage } from "@/lib/api";
import { useEffectMutation } from "@/lib/api/mutation";
import { reminderKeys, remindersApi, useNotificationSettings } from "@/lib/api/reminders";
import { useAuth } from "@/lib/auth";
import { useHasConsent } from "@/lib/consent";
import type { NotificationSettings } from "@/lib/reminders";
import { useColors } from "@/lib/theme";
import { usePullToRefresh } from "@/lib/use-pull-to-refresh";

/**
 * Settings (JAN-70): notifications per kind with the stable's reminder time, inviting (admins only), then the
 * account (name, presence, privacy, sign-out).
 */
export default function SettingsRoute() {
  return <Settings back />;
}

/** Also the "Ich" tab (`back` false): then with the links that used to be tabs or in the app menu. */
export function Settings({ back }: { back: boolean }) {
  const c = useColors();
  const queryClient = useQueryClient();
  const { user, hasStable, signOut } = useAuth();
  const settings = useNotificationSettings();
  const refresh = usePullToRefresh([settings]);
  const pushConsent = useHasConsent("push");
  const prompt = useConsentPrompt();
  const [error, setError] = useState<string | null>(null);
  const [signingOut, setSigningOut] = useState(false);

  const toggleKind = useEffectMutation({
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
    // A failure also refreshes: the switch shows what the server has.
    effects: ({ kind }) => [{ type: "notificationKind", kind }],
    errorEffects: ({ kind }) => [{ type: "notificationKind", kind }],
  });

  async function allowPush() {
    await prompt.ensure("push"); // the consent refreshes the switches (effect "consent")
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
    <Screen back={back} menu={back} refreshControl={refresh}>
      <PageHeader eyebrow={back ? undefined : "Ich"} title={back ? "Einstellungen" : (user?.name ?? "Ich")} description={user ? (back ? `${user.name}, ${user.email}` : user.email) : undefined} />

      {!back && hasStable ? (
        <RowList>
          <SettingsLink icon={Bell} title="Erinnerungen" description="Heute und diese Woche" onPress={() => router.push("/reminders" as Href)} />
          <SettingsLink icon={Activity} title="Training" description="Empfehlung, Woche, Einheit erfassen" onPress={() => router.push("/training" as Href)} />
          <SettingsLink icon={BookOpen} title="Übungen" description="Die Übungsbibliothek" onPress={() => router.push("/training/exercises" as Href)} />
          <SettingsLink icon={Horse} title="Alle Pferde" description="Pferde im Stall, Pferd anlegen" onPress={() => router.push("/horses" as Href)} />
        </RowList>
      ) : null}

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      <Section title="Benachrichtigungen">
        {pushConsent === false ? (
          <View className="gap-2 border-l-2 border-accent pl-3">
            <View className="flex-row items-center gap-2">
              <Icon as={BellOff} size={18} className="text-accent-text" />
              <Text variant="bodyStrong" className="flex-1">
                Mitteilungen sind aus
              </Text>
            </View>
            <Text variant="secondary">
              Ohne deine Erlaubnis kommen keine Mitteilungen, egal was du unten einstellst. Erinnerungen siehst du trotzdem in der Übersicht.
            </Text>
            <Button label="Erlauben" variant="outline" size="sm" onPress={() => allowPush()} />
          </View>
        ) : null}
        <WebPushCard />
        {settings.isPending ? (
          <ActivityIndicator color={c.primary.DEFAULT} className="self-start" />
        ) : settings.isError ? (
          <View className="gap-3">
            <Text variant="body" tone="danger">
              {errorMessage(settings.error)}
            </Text>
            <Button label="Erneut versuchen" variant="outline" onPress={() => settings.refetch()} />
          </View>
        ) : (
          <RowList>
            {settings.data.items.map((item) => (
              <Switch
                key={item.kind}
                label={item.label}
                description={item.description}
                value={item.enabled}
                className="py-2"
                onValueChange={(enabled) => toggleKind.mutate({ kind: item.kind, enabled })}
              />
            ))}
          </RowList>
        )}
        <ReminderTimeCard />
      </Section>

      {user?.is_admin ? (
        <Section title="Stallgasse">
          <InviteCard />
        </Section>
      ) : null}

      <Section title="Konto">
        <RowList>
          <SettingsLink
            icon={Pencil}
            title="Name"
            description={user?.name ?? "Ändern"}
            onPress={() => router.push("/(auth)/name" as Href)}
          />
          {hasStable ? (
            <SettingsLink
              icon={MapPin}
              title="Anwesenheit"
              description="Automatisch erkennen, Sichtbarkeit"
              onPress={() => router.push("/presence" as Href)}
            />
          ) : null}
          <SettingsLink
            icon={ShieldCheck}
            title="Datenschutz"
            description="Erlaubnisse, Datenexport, Konto löschen"
            onPress={() => router.push("/settings/privacy" as Href)}
          />
        </RowList>
        <Button label="Abmelden" icon={LogOut} variant="outline" fullWidth loading={signingOut} onPress={() => leave()} />
      </Section>
      {prompt.sheet}
    </Screen>
  );
}

/** A row that opens another screen: icon, title, one line of context and a chevron. */
function SettingsLink({
  icon,
  title,
  description,
  onPress,
}: {
  icon: LucideIcon;
  title: string;
  description: string;
  onPress: () => void;
}) {
  return (
    <ListRow
      leading={<Icon as={icon} size={20} className="text-muted" />}
      title={title}
      detail={description}
      accessibilityLabel={`${title}. ${description}`}
      onPress={onPress}
    />
  );
}
