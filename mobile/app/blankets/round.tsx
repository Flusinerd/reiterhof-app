import { useQueryClient } from "@tanstack/react-query";
import { Stack, useLocalSearchParams, useRouter } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { Check, ChevronLeft } from "lucide-react-native";
import { useRef, useState } from "react";
import { Pressable, View } from "react-native";

import { BlanketChoiceSheet } from "@/components/blanket-choice-sheet";
import { BlanketPhoto } from "@/components/blanket-photo";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Avatar, Button, Icon, PageHeader, Screen, Text, ThemeScope } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { blanketKeys, useSetState, useToday } from "@/lib/api/blankets";
import { useAuth } from "@/lib/auth";
import {
  boxLine,
  firstOpenIndex,
  nextIndex,
  peekText,
  previousIndex,
  roundActions,
  roundDetails,
  roundEyebrow,
  roundOrder,
  roundSegments,
  roundSummary,
  roundTitle,
  type SegmentState,
} from "@/lib/blanket-round";
import type { StateAction } from "@/lib/blankets";
import { cn } from "@/lib/cn";
import { palettes } from "@/lib/tokens";

/**
 * "Abendrunde": the blanket round of the evening, one horse per screen, always in the dark
 * ("Abendlicht"), however the phone is set. docs/domains/blankets.md, "Evening round".
 */
export default function BlanketRound() {
  return (
    <ThemeScope scheme="dark">
      <StatusBar style="light" />
      <Stack.Screen options={{ contentStyle: { backgroundColor: palettes.dark.background } }} />
      <Round />
    </ThemeScope>
  );
}

function Round() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { me } = useAuth();
  const { horse: startHorse } = useLocalSearchParams<{ horse?: string }>();
  const today = useToday();
  const setState = useSetState();
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";
  const mineIds = new Set([...(me?.roles.owned_horse_ids ?? []), ...(me?.roles.rider_horse_ids ?? [])]);

  const order = roundOrder(today.data?.horses ?? [], mineIds);
  const orderRef = useRef(order);
  orderRef.current = order;

  // undefined = not chosen yet (the first open horse, or the one of `?horse=`), null = the summary.
  const [index, setIndex] = useState<number | null | undefined>(undefined);
  const startIndex = startHorse ? order.findIndex((h) => h.horse.id === startHorse) : -1;
  const current = index === undefined ? (startIndex >= 0 ? startIndex : firstOpenIndex(order)) : index;
  const [pending, setPending] = useState<StateAction | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [choiceOpen, setChoiceOpen] = useState(false);

  const item = current === null ? null : (order[current] ?? null);

  /** Waits for the fresh state of the stable (the summary reads it), then shows the next horse or the summary. */
  async function advance(from: number) {
    await queryClient.refetchQueries({ queryKey: blanketKeys.today });
    setIndex(nextIndex(orderRef.current, from));
  }

  async function act(action: StateAction) {
    if (!item || current === null) return;
    setError(null);
    setPending(action);
    try {
      await setState.mutateAsync({ horseId: item.horse.id, action });
      await advance(current);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setPending(null);
    }
  }

  const leave = () => {
    if (router.canGoBack()) router.back();
    else router.replace("/blankets");
  };

  if (!today.data) {
    return (
      <Screen back menu={false}>
        {today.isError ? <HorseError error={today.error} onRetry={() => today.refetch()} /> : <HorseLoading />}
      </Screen>
    );
  }

  if (order.length === 0) {
    return (
      <Screen back menu={false}>
        <PageHeader eyebrow="Abendrunde" title="Keine Pferde" description="Im Stall sind noch keine Pferde angelegt." />
      </Screen>
    );
  }

  if (!item || current === null) {
    const summary = roundSummary(order, timeZone);
    return (
      <Screen back menu={false}>
        <PageHeader eyebrow="Abendrunde" title={summary.title} description={summary.subtitle} />
        <View>
          {summary.rows.map((row, i) => (
            <Pressable
              key={row.horseId}
              accessibilityRole="button"
              accessibilityLabel={`${row.name}: ${row.text}. Zum Ändern tippen`}
              onPress={() => setIndex(order.findIndex((h) => h.horse.id === row.horseId))}
              className={cn(
                "min-h-touch flex-row items-center gap-3 py-3 active:opacity-70",
                i < summary.rows.length - 1 && "border-b border-divider",
              )}
            >
              <View
                className={cn("h-6 w-6 items-center justify-center rounded-pill", row.done ? "bg-success" : "border-[1.5px] border-accent")}
              >
                {row.done ? <Icon as={Check} size={14} strokeWidth={3} className="text-primary-foreground" /> : null}
              </View>
              <View className="flex-1">
                <Text variant="bodyStrong">{row.name}</Text>
                <Text variant="bodySm" tone={row.done ? "muted" : "accent"}>
                  {row.text}
                </Text>
              </View>
              {row.meta ? (
                <Text variant="bodySm" tone="brass">
                  {row.meta}
                </Text>
              ) : null}
            </Pressable>
          ))}
        </View>
        <Button label="Fertig" size="lg" fullWidth onPress={leave} />
      </Screen>
    );
  }

  const rec = item.recommendation;
  const mine = mineIds.has(item.horse.id);
  const actions = roundActions(rec);
  const details = roundDetails(rec, today.data.weather);
  const nextItem = (() => {
    const n = nextIndex(order, current);
    return n === null ? null : order[n];
  })();
  const prev = previousIndex(current);
  const prevItem = prev === null ? null : order[prev];
  const busy = pending !== null;

  return (
    <Screen back menu={false}>
      <View className="gap-2">
        <Text variant="eyebrow" tone="brass">
          {roundEyebrow(order.length)}
        </Text>
        <Segments states={roundSegments(order, current)} />
      </View>

      <View className="gap-5 rounded-card border border-border bg-raised p-5">
        <View className="flex-row items-center gap-4">
          <Avatar name={item.horse.name} colorKey={item.horse.color_key} size="lg" />
          <View className="flex-1 gap-0.5">
            <Text variant="titleLg" accessibilityRole="header">
              {item.horse.name}
            </Text>
            {boxLine(item, mine) ? (
              <Text variant="bodySm" tone="muted">
                {boxLine(item, mine)}
              </Text>
            ) : null}
          </View>
        </View>

        <View className="flex-row items-center gap-3">
          {rec.status === "blanket" && rec.blanket?.photo_url ? <BlanketPhoto url={rec.blanket.photo_url} size={64} /> : null}
          <View className="flex-1 gap-1">
            <Text variant="eyebrow">Heute Nacht</Text>
            <Text className="font-display-medium text-title text-foreground">{roundTitle(rec)}</Text>
            {details.map((line, i) => (
              <Text key={i} variant="bodySm" tone={i === 0 ? "muted" : "default"}>
                {line}
              </Text>
            ))}
          </View>
        </View>

        <Button
          label={actions.primary.label}
          icon={Check}
          fullWidth
          className="h-[72px]"
          loading={pending === actions.primary.action}
          disabled={busy && pending !== actions.primary.action}
          onPress={() => void act(actions.primary.action)}
        />
        <View className="flex-row gap-3">
          <Button label="Andere Decke" variant="outline" className="flex-1" disabled={busy} onPress={() => setChoiceOpen(true)} />
          <Button
            label="Später"
            variant="outline"
            className="flex-1"
            disabled={busy}
            onPress={() => {
              setError(null);
              setIndex(nextIndex(order, current));
            }}
          />
        </View>
        {actions.extra.length > 0 ? (
          <View className="flex-row gap-3">
            {actions.extra.map((a) => (
              <Button
                key={a.action}
                label={a.label}
                variant="ghost"
                size="sm"
                className="flex-1"
                loading={pending === a.action}
                disabled={busy && pending !== a.action}
                onPress={() => void act(a.action)}
              />
            ))}
          </View>
        ) : null}
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
      </View>

      {nextItem ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${peekText(nextItem).title}. Jetzt dorthin springen`}
          onPress={() => setIndex(order.indexOf(nextItem))}
          className="min-h-touch flex-row items-center gap-3 opacity-60 active:opacity-90"
        >
          <Avatar name={nextItem.horse.name} colorKey={nextItem.horse.color_key} size="sm" />
          <View className="flex-1">
            <Text variant="bodyStrong" className="text-body-sm">
              {peekText(nextItem).title}
            </Text>
            <Text variant="caption">{peekText(nextItem).detail}</Text>
          </View>
        </Pressable>
      ) : null}

      {prevItem ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Zurück zu ${prevItem.horse.name}`}
          onPress={() => {
            setError(null);
            setIndex(prev);
          }}
          className="min-h-touch flex-row items-center gap-1 self-start"
        >
          <Icon as={ChevronLeft} size={16} className="text-muted" />
          <Text variant="bodySm" tone="muted">
            Zurück zu {prevItem.horse.name}
          </Text>
        </Pressable>
      ) : null}

      <BlanketChoiceSheet
        key={item.horse.id}
        open={choiceOpen}
        onOpenChange={setChoiceOpen}
        horseId={item.horse.id}
        mode="cover"
        onSaved={() => void advance(current)}
      />
    </Screen>
  );
}

/** One bar per horse: done in `success`, the current one in brass, the rest open. */
function Segments({ states }: { states: readonly SegmentState[] }) {
  const done = states.filter((s) => s === "done").length;
  return (
    <View
      className="flex-row gap-1"
      accessibilityRole="progressbar"
      accessibilityLabel={`${done} von ${states.length} versorgt`}
      accessibilityValue={{ min: 0, max: states.length, now: done }}
    >
      {states.map((s, i) => (
        <View
          key={i}
          className={cn("h-1 flex-1 rounded-pill", s === "done" ? "bg-success" : s === "current" ? "bg-brass" : "bg-border")}
        />
      ))}
    </View>
  );
}
