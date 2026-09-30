import { router, useLocalSearchParams, type Href } from "expo-router";
import { ChevronRight, ClipboardList, Pencil, Plus, Stethoscope, TriangleAlert } from "lucide-react-native";
import { useState } from "react";
import { Alert, View } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { RehaTimeline } from "@/components/reha-timeline";
import { RehaTodayHero } from "@/components/reha-today-hero";
import { Badge, Button, Card, Divider, Icon, PressableCard, Screen, SectionLabel, Text } from "@/components/ui";
import {
  rehaError,
  rehaKeys,
  useEndPlan,
  useMarkDone,
  useReha,
  useUnmarkDone,
  type RehaPlan,
  type RehaView,
} from "@/lib/api/reha";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { checkupText, dateRange, planProgress, planStatusText } from "@/lib/reha";
import { formatDate } from "@/lib/training";

function Fact({ label, value }: { label: string; value: string | null | undefined }) {
  if (!value) return null;
  return (
    <View className="gap-1">
      <Text variant="secondary">{label}</Text>
      <Text variant="body">{value}</Text>
    </View>
  );
}

function ProgressBar({ value }: { value: number }) {
  return (
    <View className="h-2 overflow-hidden rounded-pill bg-divider" accessibilityRole="progressbar" accessibilityValue={{ min: 0, max: 100, now: Math.round(value * 100) }}>
      <View className="h-2 rounded-pill bg-primary" style={{ width: `${Math.round(value * 100)}%` }} />
    </View>
  );
}

function PlanCard({ plan }: { plan: RehaPlan }) {
  return (
    <Card className="gap-4">
      <View className="gap-1">
        <Text variant="bodyStrong">{plan.diagnosis}</Text>
        <Text variant="secondary">
          {formatDate(plan.start_date)} bis {formatDate(plan.end_date)}
        </Text>
      </View>
      <View className="gap-2">
        <Text variant="bodySm">{planStatusText(plan.state, plan.day_index, plan.total_days, plan.start_date)}</Text>
        <ProgressBar value={planProgress(plan.state, plan.day_index, plan.total_days)} />
        <Text variant="caption">
          {plan.done_days.length === 1 ? "1 Tag als erledigt markiert" : `${plan.done_days.length} Tage als erledigt markiert`}
        </Text>
      </View>
      <Fact label="Tierarzt" value={plan.vet} />
    </Card>
  );
}

function CheckupCard({ plan }: { plan: RehaPlan }) {
  if (!plan.checkup_date) return null;
  const overdue = (plan.checkup_in_days ?? 0) < 0;
  return (
    <Card className="gap-2">
      <View className="flex-row items-center gap-3">
        <Icon as={Stethoscope} size={24} className="text-primary-deep" />
        <View className="flex-1">
          <Text variant="bodyStrong">{formatDate(plan.checkup_date)}</Text>
          <Text variant="secondary">{plan.vet ? `Kontrolle bei ${plan.vet}` : "Kontrolle beim Tierarzt"}</Text>
        </View>
        {plan.checkup_in_days !== null ? (
          <Badge variant={overdue ? "danger" : plan.checkup_in_days <= 2 ? "accent" : "neutral"} label={checkupText(plan.checkup_in_days)} />
        ) : null}
      </View>
      <Text variant="caption">Du bekommst zwei Tage vorher und am Morgen des Termins eine Erinnerung.</Text>
    </Card>
  );
}

function History({ plans }: { plans: RehaPlan[] }) {
  if (plans.length === 0) return null;
  return (
    <>
      <SectionLabel>Frühere Pläne</SectionLabel>
      <Card padded={false}>
        {plans.map((p, i) => (
          <View key={p.id}>
            {i > 0 ? <Divider /> : null}
            <View className="gap-1 p-4">
              <Text variant="bodyStrong">{p.diagnosis}</Text>
              <Text variant="secondary">
                {dateRange(p.start_date, p.end_date)} · {p.phases.length === 1 ? "1 Phase" : `${p.phases.length} Phasen`}
                {p.ended_on ? ` · beendet am ${formatDate(p.ended_on)}` : ""}
              </Text>
              <Text variant="caption">{p.done_days.length} Tage erledigt</Text>
            </View>
          </View>
        ))}
      </Card>
    </>
  );
}

/**
 * Reha plan of a horse (JAN-66, JAN-67): "Heute erlaubt" with the "Heute erledigt" button, the
 * phase timeline, abort criteria and the vet checkup. The owner (and admins) create and edit the
 * plan; riders see everything; other members only see today's rule.
 */
export default function Reha() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const reha = useReha(id);
  const planId = reha.data?.plan?.id ?? reha.data?.today?.plan_id ?? "";
  const markDone = useMarkDone(id ?? "", planId);
  const unmark = useUnmarkDone(id ?? "", planId);
  const end = useEndPlan(id ?? "", planId);
  const [error, setError] = useState<string | null>(null);

  useInvalidateOnEvents({ "reha.changed": [rehaKeys.all] });

  if (reha.isPending) {
    return (
      <Screen back>
        <HorseLoading />
      </Screen>
    );
  }
  if (reha.isError || !reha.data) {
    return (
      <Screen back>
        <HorseError error={reha.error} onRetry={() => void reha.refetch()} />
      </Screen>
    );
  }

  const view: RehaView = reha.data;
  const plan = view.plan;
  const go = (href: string) => router.push(href as Href);
  const fail = (e: unknown) => setError(rehaError(e));

  const confirmEnd = () => {
    Alert.alert("Reha-Plan beenden?", `Der Plan für ${view.horse_name} wird beendet und der Trainingsstatus wieder auf „fit“ gesetzt.`, [
      { text: "Abbrechen", style: "cancel" },
      { text: "Plan beenden", style: "destructive", onPress: () => end.mutate(undefined, { onError: fail }) },
    ]);
  };

  return (
    <Screen back>
      <RehaTodayHero
        view={view}
        busy={markDone.isPending || unmark.isPending}
        onDone={() => {
          setError(null);
          markDone.mutate(view.date, { onError: fail });
        }}
        onUndo={() => {
          setError(null);
          unmark.mutate(view.date, { onError: fail });
        }}
      />
      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      {view.view === "today" ? (
        <Text variant="secondary">
          Du siehst nur, was heute erlaubt ist. Den ganzen Plan sehen der Besitzer und die Reitbeteiligungen.
        </Text>
      ) : null}

      {view.view === "full" && plan ? (
        <>
          <SectionLabel>Plan</SectionLabel>
          <PlanCard plan={plan} />

          <SectionLabel>Phasen</SectionLabel>
          <RehaTimeline phases={plan.phases} />

          {plan.abort_criteria ? (
            <>
              <SectionLabel>Abbruchkriterien</SectionLabel>
              <Card className="gap-2 border-accent bg-accent-soft">
                <View className="flex-row items-center gap-2">
                  <Icon as={TriangleAlert} size={20} className="text-accent-text" />
                  <Text variant="bodyStrong" tone="accent">
                    Sofort abbrechen bei
                  </Text>
                </View>
                <Text variant="body">{plan.abort_criteria}</Text>
              </Card>
            </>
          ) : null}

          {plan.checkup_date ? (
            <>
              <SectionLabel>Tierarzt-Kontrolle</SectionLabel>
              <CheckupCard plan={plan} />
            </>
          ) : null}
        </>
      ) : null}

      {view.view === "full" ? (
        <>
          <PressableCard
            accessibilityLabel="Auffälligkeit melden"
            className="flex-row items-center gap-3"
            onPress={() => go(`/observations/new?horse=${id}`)}
          >
            <Icon as={ClipboardList} size={24} className="text-primary-deep" />
            <View className="flex-1">
              <Text variant="bodyStrong">Auffälligkeit melden</Text>
              <Text variant="secondary">Lahmheit, Schwellung oder etwas anderes bemerkt?</Text>
            </View>
            <Icon as={ChevronRight} size={20} className="text-muted" />
          </PressableCard>

          {view.can_edit ? (
            <View className="gap-3">
              {plan ? (
                <>
                  <Button label="Plan bearbeiten" variant="outline" icon={Pencil} fullWidth onPress={() => go(`/reha/edit?horse=${id}&plan=${plan.id}`)} />
                  <Button label="Plan beenden" variant="ghost" fullWidth loading={end.isPending} onPress={confirmEnd} />
                </>
              ) : (
                <Button label="Reha-Plan anlegen" icon={Plus} fullWidth onPress={() => go(`/reha/edit?horse=${id}`)} />
              )}
            </View>
          ) : null}

          <History plans={view.history} />
        </>
      ) : null}
    </Screen>
  );
}
