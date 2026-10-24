import { router, useLocalSearchParams, type Href } from "expo-router";
import { ClipboardList, Megaphone, Pencil, Plus, Stethoscope } from "lucide-react-native";
import { useState } from "react";
import { Alert, View } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { RehaTimeline } from "@/components/reha-timeline";
import { RehaTodayHead } from "@/components/reha-today-head";
import { Button, Icon, Screen, Section, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
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
import { useObservation } from "@/lib/api/observations";
import { observationRoute, observationTitle, reportedText } from "@/lib/observations";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { checkupText, dateRange, planProgress, planStatusText } from "@/lib/reha";
import { formatDate } from "@/lib/training";

function ProgressBar({ value }: { value: number }) {
  return (
    <View className="h-2 overflow-hidden rounded-pill bg-divider" accessibilityRole="progressbar" accessibilityValue={{ min: 0, max: 100, now: Math.round(value * 100) }}>
      <View className="h-2 rounded-pill bg-brass" style={{ width: `${Math.round(value * 100)}%` }} />
    </View>
  );
}

/** The plan at a glance, on the page: diagnosis, period, progress and the vet. */
function PlanBlock({ plan }: { plan: RehaPlan }) {
  return (
    <View className="gap-4">
      <View className="gap-1">
        <Text variant="title">{plan.diagnosis}</Text>
        <Text variant="bodySm" tone="muted">
          {formatDate(plan.start_date)} bis {formatDate(plan.end_date)}
        </Text>
      </View>
      <View className="gap-2">
        <Text variant="bodySm">{planStatusText(plan.state, plan.day_index, plan.total_days, plan.start_date)}</Text>
        <ProgressBar value={planProgress(plan.state, plan.day_index, plan.total_days)} />
        <Text variant="caption">
          {plan.done_days.length === 1 ? "1 Tag erledigt" : `${plan.done_days.length} Tage erledigt`}
        </Text>
      </View>
      {plan.vet ? (
        <View className="gap-1">
          <Text variant="eyebrow">Tierarzt</Text>
          <Text variant="body">{plan.vet}</Text>
        </View>
      ) : null}
    </View>
  );
}

/** The observation ("Auffälligkeit") the plan was created from (JAN-68); opens its detail screen. */
function SourceObservation({ observationId }: { observationId: string }) {
  const query = useObservation(observationId);
  const o = query.data;
  if (!o) return null;
  return (
    <ListRow
      leading={<Icon as={Megaphone} size={22} className="text-muted" />}
      title={observationTitle(o)}
      detail={`${o.reporter.name}, ${reportedText(o.created_at, new Date())}`}
      accessibilityLabel={`${observationTitle(o)} öffnen`}
      onPress={() => router.push(observationRoute(o.id) as Href)}
    />
  );
}

function CheckupRow({ plan }: { plan: RehaPlan }) {
  if (!plan.checkup_date) return null;
  const overdue = (plan.checkup_in_days ?? 0) < 0;
  return (
    <View className="gap-1">
      <ListRow
        leading={<Icon as={Stethoscope} size={22} className="text-muted" />}
        eyebrow={plan.vet ? `Kontrolle bei ${plan.vet}` : "Kontrolle beim Tierarzt"}
        title={formatDate(plan.checkup_date)}
        trailing={
          plan.checkup_in_days !== null ? (
            <Text variant="bodySm" tone={overdue ? "danger" : plan.checkup_in_days <= 2 ? "accent" : "brass"} className="font-sans-semibold">
              {checkupText(plan.checkup_in_days)}
            </Text>
          ) : null
        }
      />
      <Text variant="caption">Erinnerung zwei Tage vorher und am Termintag.</Text>
    </View>
  );
}

function History({ plans }: { plans: RehaPlan[] }) {
  if (plans.length === 0) return null;
  return (
    <Section title="Frühere Pläne">
      <RowList>
        {plans.map((p) => (
          <ListRow
            key={p.id}
            title={p.diagnosis}
            detail={`${dateRange(p.start_date, p.end_date)} · ${p.phases.length === 1 ? "1 Phase" : `${p.phases.length} Phasen`}${p.ended_on ? ` · beendet am ${formatDate(p.ended_on)}` : ""} · ${p.done_days.length} Tage erledigt`}
            accessibilityLabel={`${p.diagnosis}, ${dateRange(p.start_date, p.end_date)}`}
          />
        ))}
      </RowList>
    </Section>
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
    Alert.alert("Reha-Plan beenden?", `Der Trainingsstatus von ${view.horse_name} wird wieder „fit“.`, [
      { text: "Abbrechen", style: "cancel" },
      { text: "Plan beenden", style: "destructive", onPress: () => end.mutate(undefined, { onError: fail }) },
    ]);
  };

  return (
    <Screen back>
      <RehaTodayHead
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
          Den ganzen Plan sehen nur Besitzer und Reitbeteiligungen.
        </Text>
      ) : null}

      {view.view === "full" && plan ? (
        <>
          <Section title="Plan">
            <PlanBlock plan={plan} />
          </Section>

          {plan.observation_id ? (
            <Section title="Ausgelöst durch">
              <SourceObservation observationId={plan.observation_id} />
            </Section>
          ) : null}

          <Section title="Phasen">
            <RehaTimeline phases={plan.phases} />
          </Section>

          {plan.abort_criteria ? (
            <Section title="Abbruchkriterien">
              <View className="gap-1 border-l-2 border-accent pl-3">
                <Text variant="eyebrow" tone="accent">
                  Sofort abbrechen bei
                </Text>
                <Text variant="body">{plan.abort_criteria}</Text>
              </View>
            </Section>
          ) : null}

          {plan.checkup_date ? (
            <Section title="Tierarzt-Kontrolle">
              <CheckupRow plan={plan} />
            </Section>
          ) : null}
        </>
      ) : null}

      {view.view === "full" ? (
        <>
          <ListRow
            leading={<Icon as={ClipboardList} size={22} className="text-muted" />}
            title="Auffälligkeit melden"
            detail="Lahmheit, Schwellung oder etwas anderes bemerkt?"
            accessibilityLabel="Auffälligkeit melden"
            onPress={() => go(`/observations/new?horse=${id}`)}
          />

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
