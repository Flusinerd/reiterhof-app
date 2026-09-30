import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalSearchParams } from "expo-router";
import { Calendar, Check, CheckCheck, HandHeart, Heart, Repeat, Square, SquareCheck, UserMinus, X } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, Pressable, RefreshControl, View } from "react-native";

import { RequestTypeIcon } from "@/components/request-type-icon";
import { Avatar, Badge, Button, Card, Divider, Icon, PageHeader, Pill, Screen, Section, Sheet, Text } from "@/components/ui";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import { useAuth } from "@/lib/auth";
import { colors } from "@/lib/theme";
import {
  acceptLabel,
  describeRule,
  currentReminder,
  formatInstant,
  formatWhen,
  helperCountText,
  helperStatusText,
  payloadDetails,
  reminderAt,
  reminderChoices,
  requestTitle,
  statusBadge,
  taskChips,
  type HelpRequest,
  type ReminderOption,
} from "@/lib/requests";
import { addToDeviceCalendar } from "@/lib/requests-calendar";
import { requestErrorMessage } from "@/lib/requests-errors";

function Row({ label, value }: { label: string; value: string }) {
  return (
    <View className="flex-row justify-between gap-4">
      <Text variant="secondary">{label}</Text>
      <Text variant="bodySm" className="flex-1 text-right">
        {value}
      </Text>
    </View>
  );
}

export default function RequestDetail() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [checked, setChecked] = useState<Record<string, boolean>>({});

  const query = useQuery({ queryKey: requestKeys.detail(id), queryFn: () => requestsApi.get(id) });
  const r = query.data;

  function afterChange(updated: HelpRequest) {
    queryClient.setQueryData(requestKeys.detail(id), updated);
    void queryClient.invalidateQueries({ queryKey: requestKeys.all });
  }

  const action = useMutation({
    mutationFn: (fn: () => Promise<HelpRequest>) => fn(),
    onMutate: () => {
      setError(null);
      setNotice(null);
    },
    onSuccess: afterChange,
    onError: (e) => setError(requestErrorMessage(e)),
  });

  const remind = useMutation({
    mutationFn: (value: ReminderOption) => requestsApi.setReminder(id, reminderAt(value, r?.date ?? "", r?.time_from ?? null)),
    onMutate: () => setError(null),
    onSuccess: afterChange,
    onError: (e) => setError(requestErrorMessage(e)),
  });

  const thank = useMutation({
    mutationFn: (userId: string) => requestsApi.thank(id, userId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: requestKeys.all }),
    onError: (e) => setError(requestErrorMessage(e)),
  });

  if (query.isPending) {
    return (
      <Screen back>
        <ActivityIndicator color={colors.primary.DEFAULT} />
      </Screen>
    );
  }
  if (!r) {
    return (
      <Screen back>
        <PageHeader title="Anfrage nicht gefunden" description={requestErrorMessage(query.error)} />
        <Button label="Erneut versuchen" variant="outline" onPress={() => void query.refetch()} />
      </Screen>
    );
  }

  const active = r.status === "open" || r.status === "assigned";
  const canManage = r.is_creator || !!user?.is_admin;
  const status = statusBadge(r.status);
  const chips = taskChips(r);
  const rule = describeRule(r.recurring_rule, r.date);
  const details = payloadDetails(r);
  const busy = action.isPending;

  async function exportToCalendar() {
    if (!r) return;
    const result = await addToDeviceCalendar(r);
    setNotice(result === "failed" ? "Das hat nicht geklappt." : null);
  }

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={colors.primary.DEFAULT} />
      }
    >
      <PageHeader
        eyebrow={`Von ${r.is_creator ? "dir" : r.creator_name}`}
        title={requestTitle(r)}
        description={formatWhen(r)}
        action={<RequestTypeIcon type={r.type} size={44} />}
      >
        <View className="flex-row flex-wrap items-center gap-3">
          <Badge label={status.label} variant={status.variant} />
          <Text variant="secondary">{helperCountText(r.helpers_count, r.helpers_needed, r.type)}</Text>
        </View>
      </PageHeader>

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      {notice ? <Text variant="bodySm">{notice}</Text> : null}

      {r.can_accept ? (
        <Button
          label={acceptLabel(r.type)}
          icon={HandHeart}
          size="lg"
          fullWidth
          loading={busy}
          onPress={() => action.mutate(() => requestsApi.accept(r.id))}
        />
      ) : null}

      <Section title="Details">
        <Card className="gap-3">
          <Row label="Wann" value={formatWhen(r)} />
          {r.location ? <Row label="Wo" value={r.location} /> : null}
          {r.horse_name ? <Row label="Pferd" value={r.horse_name} /> : null}
          {details.map((d) => (
            <Row key={d.label} label={d.label} value={d.value} />
          ))}
          {rule ? (
            <View className="flex-row items-center justify-between gap-4">
              <Text variant="secondary">Wiederholung</Text>
              <View className="flex-row items-center gap-1.5">
                <Icon as={Repeat} size={16} className="text-muted" />
                <Text variant="bodySm">{rule}</Text>
              </View>
            </View>
          ) : null}
          {r.description ? (
            <>
              <Divider />
              <Text variant="body">{r.description}</Text>
            </>
          ) : null}
        </Card>
      </Section>

      {r.is_helper && active ? (
        <Section title="Meine Erinnerung" description="Nur du siehst und erhältst sie.">
          <View className="flex-row flex-wrap gap-2">
            {reminderChoices(r.date, r.time_from).map((o) => (
              <Pill
                key={o.value}
                label={o.label}
                selected={currentReminder(r.my_remind_at, r.date, r.time_from) === o.value}
                onPress={() => remind.mutate(o.value)}
              />
            ))}
          </View>
          {currentReminder(r.my_remind_at, r.date, r.time_from) === null && r.my_remind_at ? (
            <Text variant="secondary">Aktuell: {formatInstant(r.my_remind_at)}</Text>
          ) : null}
        </Section>
      ) : null}

      {chips.length > 0 ? (
        <Section title="Checkliste">
          <Card className="gap-1 py-2">
            {chips.map((c) => (
              <Pressable
                key={c}
                accessibilityRole="checkbox"
                accessibilityState={{ checked: !!checked[c] }}
                onPress={() => setChecked({ ...checked, [c]: !checked[c] })}
                className="min-h-touch flex-row items-center gap-3"
              >
                <Icon as={checked[c] ? SquareCheck : Square} size={22} className={checked[c] ? "text-primary" : "text-muted"} />
                <Text variant="body" className={checked[c] ? "text-muted line-through" : undefined}>
                  {c}
                </Text>
              </Pressable>
            ))}
          </Card>
        </Section>
      ) : null}

      <Section title="Helfer" description={helperStatusText(r.helpers_count, r.helpers_needed, r.type)}>
        <Card className="gap-3">
          {r.helpers.length === 0 ? <Text variant="secondary">Noch niemand.</Text> : null}
          {r.helpers.map((h) => (
            <View key={h.user_id} className="flex-row items-center gap-3">
              <Avatar name={h.name} colorKey={h.avatar_color} />
              <Text variant="body" className="flex-1">
                {h.user_id === user?.id ? `${h.name} (du)` : h.name}
              </Text>
              {h.thanked ? (
                <Badge label="Bedankt" variant="primary" />
              ) : r.is_creator && r.status !== "cancelled" ? (
                <Pill label="Danke" icon={Heart} onPress={() => thank.mutate(h.user_id)} />
              ) : null}
            </View>
          ))}
        </Card>
      </Section>

      <View className="gap-3">
        {r.is_helper && active ? (
          <Button
            label="Abmelden"
            icon={UserMinus}
            variant="outline"
            fullWidth
            loading={busy}
            onPress={() => action.mutate(() => requestsApi.withdraw(r.id))}
          />
        ) : null}
        {(r.is_creator || r.is_helper) && active ? (
          <Button
            label="Erledigt"
            icon={CheckCheck}
            variant={r.can_accept ? "outline" : "primary"}
            fullWidth
            loading={busy}
            onPress={() => action.mutate(() => requestsApi.done(r.id))}
          />
        ) : null}
        {r.status !== "cancelled" && (r.is_creator || r.is_helper) ? (
          <Button label="In Kalender" icon={Calendar} variant="outline" fullWidth onPress={() => void exportToCalendar()} />
        ) : null}
        {canManage && active ? (
          <Button label="Anfrage absagen" icon={X} variant="ghost" fullWidth onPress={() => setCancelOpen(true)} />
        ) : null}
        {r.status === "done" ? (
          <View className="flex-row items-center gap-2">
            <Icon as={Check} size={18} className="text-primary" />
            <Text variant="secondary">Erledigt. Danke an alle Helfer.</Text>
          </View>
        ) : null}
      </View>

      <Sheet
        open={cancelOpen}
        onOpenChange={setCancelOpen}
        title="Anfrage absagen?"
        description={r.helpers_count > 0 ? "Die Helfer bekommen eine Nachricht." : undefined}
      >
        <View className="gap-3">
          <Button
            label={r.series_id ? "Nur diese absagen" : "Anfrage absagen"}
            variant="danger"
            fullWidth
            onPress={() => {
              setCancelOpen(false);
              action.mutate(() => requestsApi.cancel(r.id, "one"));
            }}
          />
          {r.series_id ? (
            <Button
              label="Ganze Serie absagen"
              variant="outline"
              fullWidth
              onPress={() => {
                setCancelOpen(false);
                action.mutate(() => requestsApi.cancel(r.id, "series"));
              }}
            />
          ) : null}
          <Button label="Zurück" variant="ghost" fullWidth onPress={() => setCancelOpen(false)} />
        </View>
      </Sheet>
    </Screen>
  );
}
