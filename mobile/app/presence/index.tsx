import { LogIn, LogOut } from "lucide-react-native";
import { RefreshControl, View } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { usePresenceCheckIn } from "@/components/presence-check-in";
import { Avatar, Button, Card, Divider, PageHeader, Screen, Section, Switch, Text, ToggleGroup, ToggleGroupItem } from "@/components/ui";
import { colors } from "@/lib/theme";
import { errorMessage, type PresenceVisibility } from "@/lib/api";
import { useAuth } from "@/lib/auth";
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

export default function PresenceScreen() {
  const { me } = useAuth();
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

  let heroDescription: string | undefined;
  if (open) heroDescription = `Andere sehen dich ${visibility === "all" ? sinceLabel(open.arrived_at, timeZone) : "nach deiner Einstellung"}.`;
  else if (data?.me.last_visit?.left_at) {
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
        <RefreshControl refreshing={overview.isRefetching} onRefresh={() => void overview.refetch()} tintColor={colors.primary.DEFAULT} />
      }
    >
      <PageHeader
        title={open ? "Im Stall" : "Nicht im Stall"}
        value={open ? formatClock(open.arrived_at, timeZone) : undefined}
        valueSize="sm"
        unit={open ? "Uhr" : undefined}
        description={heroDescription}
      >
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

      <Section title="Jetzt da">
        {overview.isError ? (
          <Text variant="body" tone="danger">
            {errorMessage(overview.error)}
          </Text>
        ) : data && data.here.length > 0 ? (
          <View className="flex-row flex-wrap gap-3">
            {data.here.map((p) => (
              <Card key={p.user_id} shape="tile" className="w-[47%] items-center gap-2 px-3 py-4">
                <Avatar name={p.name} colorKey={p.avatar_color} size="lg" />
                <Text variant="bodyStrong" numberOfLines={1}>
                  {p.name}
                </Text>
                <Text variant="secondary">{sinceLabel(p.since, timeZone)}</Text>
              </Card>
            ))}
          </View>
        ) : (
          <Text variant="body" tone="muted">
            {data ? "Sonst niemand da." : "Wird geladen ..."}
          </Text>
        )}
      </Section>

      <Section title="Zuletzt gesehen">
        {data && data.recent.length > 0 ? (
          <Card padded={false}>
            {data.recent.map((p, i) => {
              const hint = usualArrivalLabel(p.usual_arrival_hour);
              return (
                <View key={p.user_id}>
                  {i > 0 ? <Divider /> : null}
                  <View className="min-h-touch flex-row items-center gap-3 px-5 py-3">
                    <Avatar name={p.name} colorKey={p.avatar_color} size="md" />
                    <View className="flex-1">
                      <Text variant="bodyStrong" numberOfLines={1}>
                        {p.name}
                      </Text>
                      <Text variant="secondary">{lastSeenLabel(p, today, timeZone)}</Text>
                      {hint ? <Text variant="caption">{hint}</Text> : null}
                    </View>
                  </View>
                </View>
              );
            })}
          </Card>
        ) : (
          <Text variant="body" tone="muted">
            {data ? "Noch keine Besuche." : "Wird geladen ..."}
          </Text>
        )}
      </Section>

      <Section title="Einstellungen">
        <Card className="gap-4">
        {geofence.supported ? (
          <>
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
            <Text variant="caption">
              Dein Standort bleibt auf dem Handy. Der Server erfährt nur Ankunft und Abgang.
            </Text>
            <Divider />
          </>
        ) : isWeb ? (
          <>
            <Text variant="caption">Automatisches Ein- und Auschecken gibt es nur in der App.</Text>
            <Divider />
          </>
        ) : null}
        <View className="gap-2">
          <Text variant="bodyStrong">Wer sieht mich?</Text>
          <ToggleGroup
            type="single"
            value={visibility}
            onValueChange={(v) => presence.changeVisibility(v as PresenceVisibility)}
          >
            {VISIBILITY_OPTIONS.map((o) => (
              <ToggleGroupItem key={o.value} value={o.value} label={o.label} disabled={presence.changingVisibility} />
            ))}
          </ToggleGroup>
          <Text variant="secondary">{visibilityDescription(visibility)}</Text>
        </View>
        </Card>
      </Section>
      {consent.sheet}
      {presence.sheet}
    </Screen>
  );
}
