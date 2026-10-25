import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalSearchParams } from "expo-router";
import { Calendar, Check, CheckCheck, HandHeart, Heart, Square, SquareCheck, UserMinus, X, type LucideIcon } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, Pressable, RefreshControl, View } from "react-native";

import { Avatar, Badge, Button, Icon, PageHeader, Pill, Screen, Section, Sheet, Text } from "@/components/ui";
import { useEffectMutation } from "@/lib/api/mutation";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { useColors } from "@/lib/theme";
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

/** A fact of the request: the label in the eyebrow style on the left, the value on the right, a hairline below. */
function DetailRow({ label, value, last }: { label: string; value: string; last: boolean }) {
  return (
    <View className={cn("min-h-touch flex-row items-center justify-between gap-4 py-3", !last && "border-b border-divider")}>
      <Text variant="eyebrow">{label}</Text>
      <Text variant="bodySm" className="flex-1 text-right">
        {value}
      </Text>
    </View>
  );
}

/** A quiet action as a row: icon and label, the destructive one in red. */
function ActionRow({
  icon,
  label,
  tone = "default",
  loading,
  last,
  onPress,
}: {
  icon: LucideIcon;
  label: string;
  tone?: "default" | "danger";
  loading?: boolean;
  last: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ busy: !!loading, disabled: !!loading }}
      disabled={loading}
      onPress={onPress}
      className={cn("min-h-touch flex-row items-center gap-3 py-3 active:opacity-70", !last && "border-b border-divider", loading && "opacity-50")}
    >
      <Icon as={icon} size={20} className={tone === "danger" ? "text-danger" : "text-muted"} />
      <Text variant="body" tone={tone === "danger" ? "danger" : "default"}>
        {label}
      </Text>
    </Pressable>
  );
}

export default function RequestDetail() {
  const c = useColors();
  const { id } = useLocalSearchParams<{ id: string }>();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [checked, setChecked] = useState<Record<string, boolean>>({});

  const query = useQuery({ queryKey: requestKeys.detail(id), queryFn: () => requestsApi.get(id) });
  const r = query.data;

  /** The answer of the server is the new detail; the lists and the reminder center follow (effect "request"). */
  function storeDetail(updated: HelpRequest) {
    queryClient.setQueryData(requestKeys.detail(id), updated);
  }

  const action = useEffectMutation({
    mutationFn: (fn: () => Promise<HelpRequest>) => fn(),
    onMutate: () => {
      setError(null);
      setNotice(null);
    },
    onSuccess: storeDetail,
    onError: (e) => setError(requestErrorMessage(e)),
    effects: () => [{ type: "request" }],
    errorEffects: () => [{ type: "request" }], // a conflict (full, cancelled) means the view was stale
  });

  const remind = useEffectMutation({
    mutationFn: (value: ReminderOption) => requestsApi.setReminder(id, reminderAt(value, r?.date ?? "", r?.time_from ?? null)),
    onMutate: () => setError(null),
    onSuccess: storeDetail,
    onError: (e) => setError(requestErrorMessage(e)),
    effects: () => [{ type: "request" }],
  });

  const thank = useEffectMutation({
    mutationFn: (userId: string) => requestsApi.thank(id, userId),
    onError: (e) => setError(requestErrorMessage(e)),
    effects: () => [{ type: "request" }],
  });

  if (query.isPending) {
    return (
      <Screen back>
        <ActivityIndicator color={c.primary.DEFAULT} />
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
  const involved = r.is_creator || r.is_helper;
  const canManage = r.is_creator || !!user?.is_admin;
  const status = statusBadge(r.status);
  const chips = taskChips(r);
  const rule = describeRule(r.recurring_rule, r.date);
  const busy = action.isPending;

  // The facts as rows (the time is in the head): where, which horse, what the type adds, the repeat.
  const facts: { label: string; value: string }[] = [
    ...(r.location ? [{ label: "Wo", value: r.location }] : []),
    ...(r.horse_name ? [{ label: "Pferd", value: r.horse_name }] : []),
    ...payloadDetails(r),
    ...(rule ? [{ label: "Wiederholung", value: rule }] : []),
  ];

  // The one primary action of the page: accept, else mark as done.
  const primary = r.can_accept
    ? { label: acceptLabel(r.type), icon: HandHeart, run: () => action.mutate(() => requestsApi.accept(r.id)) }
    : involved && active
      ? { label: "Erledigt", icon: CheckCheck, run: () => action.mutate(() => requestsApi.done(r.id)) }
      : null;

  const quiet: { icon: LucideIcon; label: string; tone?: "danger"; run: () => void }[] = [
    ...(r.is_helper && active
      ? [{ icon: UserMinus, label: "Abmelden", run: () => action.mutate(() => requestsApi.withdraw(r.id)) }]
      : []),
    ...(r.status !== "cancelled" && involved ? [{ icon: Calendar, label: "In Kalender", run: () => void exportToCalendar() }] : []),
    ...(canManage && active ? [{ icon: X, label: "Anfrage absagen", tone: "danger" as const, run: () => setCancelOpen(true) }] : []),
  ];

  async function exportToCalendar() {
    if (!r) return;
    const result = await addToDeviceCalendar(r);
    setNotice(result === "failed" ? "Das hat nicht geklappt." : null);
  }

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={c.primary.DEFAULT} />
      }
    >
      <PageHeader
        eyebrow={`Von ${r.is_creator ? "dir" : r.creator_name}`}
        title={requestTitle(r)}
        description={formatWhen(r)}
        action={<Avatar name={r.creator_name} colorKey={r.creator_color} size="md" />}
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

      {primary ? <Button label={primary.label} icon={primary.icon} size="lg" fullWidth loading={busy} onPress={primary.run} /> : null}

      {r.description ? (
        <Text variant="body" className="font-display-italic text-foreground/80">
          {`„${r.description}“`}
        </Text>
      ) : null}

      <View>
        {facts.map((f, i) => (
          <DetailRow key={f.label} label={f.label} value={f.value} last={i === facts.length - 1} />
        ))}
      </View>

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
          <View>
            {chips.map((chip, i) => (
              <Pressable
                key={chip}
                accessibilityRole="checkbox"
                accessibilityState={{ checked: !!checked[chip] }}
                onPress={() => setChecked({ ...checked, [chip]: !checked[chip] })}
                className={cn("min-h-touch flex-row items-center gap-3 py-2", i < chips.length - 1 && "border-b border-divider")}
              >
                <Icon as={checked[chip] ? SquareCheck : Square} size={22} className={checked[chip] ? "text-success" : "text-muted"} />
                <Text variant="body" className={checked[chip] ? "text-muted line-through" : undefined}>
                  {chip}
                </Text>
              </Pressable>
            ))}
          </View>
        </Section>
      ) : null}

      <Section title="Helfer" description={helperStatusText(r.helpers_count, r.helpers_needed, r.type)}>
        {r.helpers.length === 0 ? (
          <Text variant="body" tone="muted">
            Noch niemand.
          </Text>
        ) : (
          <View>
            {r.helpers.map((h, i) => (
              <View
                key={h.user_id}
                className={cn("min-h-[60px] flex-row items-center gap-3 py-2", i < r.helpers.length - 1 && "border-b border-divider")}
              >
                <Avatar name={h.name} colorKey={h.avatar_color} />
                <Text variant="body" className="flex-1">
                  {h.user_id === user?.id ? `${h.name} (du)` : h.name}
                </Text>
                {h.thanked ? (
                  <Text variant="bodySm" tone="brass" className="font-sans-semibold">
                    Bedankt
                  </Text>
                ) : r.is_creator && r.status !== "cancelled" ? (
                  <Pill label="Danke" icon={Heart} onPress={() => thank.mutate(h.user_id)} />
                ) : null}
              </View>
            ))}
          </View>
        )}
      </Section>

      {quiet.length > 0 ? (
        <View>
          {quiet.map((q, i) => (
            <ActionRow
              key={q.label}
              icon={q.icon}
              label={q.label}
              tone={q.tone}
              loading={busy && q.label === "Abmelden"}
              last={i === quiet.length - 1}
              onPress={q.run}
            />
          ))}
        </View>
      ) : null}

      {r.status === "done" ? (
        <View className="flex-row items-center gap-2">
          <Icon as={Check} size={18} className="text-success" />
          <Text variant="secondary">Erledigt. Danke an alle Helfer.</Text>
        </View>
      ) : null}

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
