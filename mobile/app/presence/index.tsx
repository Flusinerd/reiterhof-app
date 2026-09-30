import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { LogIn, LogOut } from "lucide-react-native";
import { useState } from "react";
import { RefreshControl, View } from "react-native";

import { Avatar, Button, Card, Divider, Hero, Screen, SectionLabel, Switch, Text, ToggleGroup, ToggleGroupItem } from "@/components/ui";
import { errorMessage, api, type PresenceVisibility } from "@/lib/api";
import { PRESENCE_EVENT, PRESENCE_KEY, presenceApi } from "@/lib/api/presence";
import { ME_KEY, useAuth } from "@/lib/auth";
import { failureMessage } from "@/lib/geofence-core";
import { useGeofence } from "@/lib/geofence";
import {
  formatClock,
  lastSeenLabel,
  localDate,
  sinceLabel,
  usualArrivalLabel,
  VISIBILITY_OPTIONS,
  visibilityDescription,
} from "@/lib/presence-format";
import { useInvalidateOnEvents } from "@/lib/realtime";

export default function PresenceScreen() {
  const { me, user } = useAuth();
  const queryClient = useQueryClient();
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";
  const [actionError, setActionError] = useState<string | null>(null);

  const overview = useQuery({ queryKey: PRESENCE_KEY, queryFn: presenceApi.overview });
  useInvalidateOnEvents({ [PRESENCE_EVENT]: [PRESENCE_KEY] });

  const toggleVisit = useMutation({
    mutationFn: (arrive: boolean) => (arrive ? presenceApi.checkIn("manual") : presenceApi.checkOut()),
    onMutate: () => setActionError(null),
    onError: (err) => setActionError(errorMessage(err)),
    onSettled: () => queryClient.invalidateQueries({ queryKey: PRESENCE_KEY }),
  });

  const changeVisibility = useMutation({
    mutationFn: (visibility: PresenceVisibility) => api.updateMe({ presence_visibility: visibility }),
    onMutate: () => setActionError(null),
    onSuccess: (updated) => queryClient.setQueryData(ME_KEY, updated),
    onError: (err) => setActionError(errorMessage(err)),
    onSettled: () => queryClient.invalidateQueries({ queryKey: PRESENCE_KEY }),
  });

  const geofence = useGeofence(me?.stable);

  const data = overview.data;
  const open = data?.me.open_visit ?? null;
  const visibility = (user?.presence_visibility ?? data?.me.visibility ?? "all") as PresenceVisibility;
  const today = localDate(new Date(), timeZone);

  let heroDescription = "Tippe auf „Bin da“, wenn du im Stall ankommst.";
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
      refreshControl={<RefreshControl refreshing={overview.isRefetching} onRefresh={() => void overview.refetch()} />}
    >
      <Hero
        eyebrow="Anwesenheit"
        title={open ? "Du bist im Stall" : "Du bist nicht im Stall"}
        value={open ? formatClock(open.arrived_at, timeZone) : undefined}
        valueSize="sm"
        unit={open ? "Uhr, seit Ankunft" : undefined}
        description={heroDescription}
      >
        <Button
          label={open ? "Ich gehe" : "Bin da"}
          icon={open ? LogOut : LogIn}
          variant="secondary"
          size="lg"
          fullWidth
          loading={toggleVisit.isPending}
          disabled={overview.isPending}
          onPress={() => toggleVisit.mutate(!open)}
        />
        {actionError ? (
          <Text variant="bodySm" tone="inverse">
            {actionError}
          </Text>
        ) : null}
      </Hero>

      <SectionLabel>Einstellungen</SectionLabel>
      <Card className="gap-4">
        {geofence.supported ? (
          <>
            <Switch
              label="Automatisch erkennen"
              description="Die App meldet dich an und ab, wenn du den Stall erreichst oder verlässt. Nur auf diesem Gerät."
              value={geofence.enabled}
              disabled={geofence.busy}
              onValueChange={(on) => void geofence.toggle(on)}
            />
            {geofence.failure ? (
              <Text variant="secondary" tone="danger">
                {failureMessage(geofence.failure)}
              </Text>
            ) : null}
            <Text variant="caption">
              Die Datenschutz-Einwilligung folgt in einer späteren Version. Dein Standort bleibt auf dem Handy, der
              Server erfährt nur „angekommen“ und „gegangen“.
            </Text>
            <Divider />
          </>
        ) : null}
        <View className="gap-2">
          <Text variant="bodyStrong">Wer sieht mich?</Text>
          <ToggleGroup
            type="single"
            value={visibility}
            onValueChange={(v) => changeVisibility.mutate(v as PresenceVisibility)}
          >
            {VISIBILITY_OPTIONS.map((o) => (
              <ToggleGroupItem key={o.value} value={o.value} label={o.label} disabled={changeVisibility.isPending} />
            ))}
          </ToggleGroup>
          <Text variant="secondary">{visibilityDescription(visibility)}</Text>
        </View>
      </Card>

      <SectionLabel>Jetzt da</SectionLabel>
      {overview.isError ? (
        <Card>
          <Text variant="body" tone="danger">
            {errorMessage(overview.error)}
          </Text>
        </Card>
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
        <Card>
          <Text variant="body" tone="muted">
            {data ? "Gerade ist niemand sonst im Stall." : "Wird geladen ..."}
          </Text>
        </Card>
      )}

      <SectionLabel>Zuletzt gesehen</SectionLabel>
      <Card padded={false}>
        {data && data.recent.length > 0 ? (
          data.recent.map((p, i) => {
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
          })
        ) : (
          <View className="p-5">
            <Text variant="body" tone="muted">
              {data ? "Noch niemand war im Stall." : "Wird geladen ..."}
            </Text>
          </View>
        )}
      </Card>
    </Screen>
  );
}
