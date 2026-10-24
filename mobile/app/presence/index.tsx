import { LogIn, LogOut } from "lucide-react-native";
import { RefreshControl, View } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { usePresenceCheckIn } from "@/components/presence-check-in";
import { Avatar, Button, PageHeader, Screen, Section, Switch, Text, ToggleGroup, ToggleGroupItem } from "@/components/ui";
import { useColors } from "@/lib/theme";
import { errorMessage, type PresenceVisibility } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { failureMessage } from "@/lib/geofence-core";
import { useGeofence } from "@/lib/geofence";
import { isWeb } from "@/lib/platform";
import {
  formatClock,
  lastSeenLabel,
  localDate,
  sinceLabel,
  usualArrivalLabel,
  VISIBILITY_OPTIONS,
  visibilityDescription,
} from "@/lib/presence-format";
import { herePeople, presenceSentence } from "@/lib/stable-tab";

/** A line of a list with hairlines: avatar, name with a detail line, something short on the right. */
function PersonRow({
  name,
  colorKey,
  detail,
  hint,
  right,
  last,
}: {
  name: string;
  colorKey: string | null;
  detail?: string;
  hint?: string | null;
  right?: string;
  last: boolean;
}) {
  return (
    <View className={cn("min-h-[60px] flex-row items-center gap-3 py-3", !last && "border-b border-divider")}>
      <Avatar name={name} colorKey={colorKey} size="md" />
      <View className="flex-1 gap-0.5">
        <Text variant="bodyStrong" numberOfLines={1}>
          {name}
        </Text>
        {detail ? <Text variant="secondary">{detail}</Text> : null}
        {hint ? <Text variant="caption">{hint}</Text> : null}
      </View>
      {right ? (
        <Text variant="bodySm" tone="brass" className="font-sans-semibold">
          {right}
        </Text>
      ) : null}
    </View>
  );
}

/** Presence: who is at the stable, my own check in and out, who was here lately, and the settings. */
export default function PresenceScreen() {
  const c = useColors();
  const { me, user } = useAuth();
  const consent = useConsentPrompt();
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";
  const presence = usePresenceCheckIn();
  const { overview, visibility, confirmPresenceSharing } = presence;

  const geofence = useGeofence(me?.stable);

  // The geofence needs the location consent first; the server refuses geofence check-ins without it.
  async function toggleGeofence(on: boolean) {
    if (on) {
      if (!(await consent.ensure("location_geofence"))) return;
      await confirmPresenceSharing();
    }
    void geofence.toggle(on);
  }

  const data = overview.data;
  const open = data?.me.open_visit ?? null;
  const today = localDate(new Date(), timeZone);
  const people = herePeople(data, user);

  let heroDescription: string | undefined;
  if (open) {
    heroDescription = `Seit ${formatClock(open.arrived_at, timeZone)} Uhr im Stall. Andere sehen dich ${
      visibility === "all" ? sinceLabel(open.arrived_at, timeZone) : "nach deiner Einstellung"
    }.`;
  } else if (data?.me.last_visit?.left_at) {
    heroDescription = `Zuletzt im Stall: ${lastSeenLabel(
      { last_seen_date: localDate(data.me.last_visit.left_at, timeZone), last_seen_at: data.me.last_visit.left_at },
      today,
      timeZone,
    )}.`;
  }

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl refreshing={overview.isRefetching} onRefresh={() => void overview.refetch()} tintColor={c.primary.DEFAULT} />
      }
    >
      <PageHeader eyebrow="Anwesenheit" title={presenceSentence(people, !!data)} description={heroDescription}>
        <Button
          label={open ? "Ich gehe" : "Bin da"}
          icon={open ? LogOut : LogIn}
          variant={open ? "secondary" : "primary"}
          size="lg"
          fullWidth
          loading={presence.toggling}
          disabled={overview.isPending}
          onPress={() => void presence.arriveOrLeave(!open)}
        />
        {presence.error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {presence.error}
          </Text>
        ) : null}
      </PageHeader>

      {overview.isError ? (
        <Text variant="body" tone="danger">
          {errorMessage(overview.error)}
        </Text>
      ) : data && data.here.length > 0 ? (
        <Section title="Jetzt da">
          <View>
            {data.here.map((p, i) => (
              <PersonRow
                key={p.user_id}
                name={p.name}
                colorKey={p.avatar_color}
                right={p.since ? sinceLabel(p.since, timeZone) : "heute da"}
                last={i === data.here.length - 1}
              />
            ))}
          </View>
        </Section>
      ) : null}

      <Section title="Zuletzt gesehen">
        {data && data.recent.length > 0 ? (
          <View>
            {data.recent.map((p, i) => (
              <PersonRow
                key={p.user_id}
                name={p.name}
                colorKey={p.avatar_color}
                detail={lastSeenLabel(p, today, timeZone)}
                hint={usualArrivalLabel(p.usual_arrival_hour)}
                last={i === data.recent.length - 1}
              />
            ))}
          </View>
        ) : (
          <Text variant="body" tone="muted">
            {data ? "Noch keine Besuche." : "Wird geladen ..."}
          </Text>
        )}
      </Section>

      <Section title="Einstellungen">
        <View className="gap-5">
          {geofence.supported ? (
            <View className="gap-3 border-b border-divider pb-5">
              <Switch
                label="Automatisch erkennen"
                description="Meldet dich automatisch an und ab. Nur auf diesem Gerät."
                value={geofence.enabled}
                disabled={geofence.busy}
                onValueChange={(on) => void toggleGeofence(on)}
              />
              {geofence.failure ? (
                <Text variant="secondary" tone="danger">
                  {failureMessage(geofence.failure)}
                </Text>
              ) : null}
              <Text variant="caption">Dein Standort bleibt auf dem Handy. Der Server erfährt nur Ankunft und Abgang.</Text>
            </View>
          ) : isWeb ? (
            <Text variant="caption" className="border-b border-divider pb-5">
              Automatisches Ein- und Auschecken gibt es nur in der App.
            </Text>
          ) : null}
          <View className="gap-2">
            <Text variant="eyebrow">Wer sieht mich?</Text>
            <ToggleGroup type="single" value={visibility} onValueChange={(v) => presence.changeVisibility(v as PresenceVisibility)}>
              {VISIBILITY_OPTIONS.map((o) => (
                <ToggleGroupItem key={o.value} value={o.value} label={o.label} disabled={presence.changingVisibility} />
              ))}
            </ToggleGroup>
            <Text variant="secondary">{visibilityDescription(visibility)}</Text>
          </View>
        </View>
      </Section>
      {consent.sheet}
      {presence.sheet}
    </Screen>
  );
}
