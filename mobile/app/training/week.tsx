import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { BookOpen, ChevronLeft, ChevronRight, Sparkles } from "lucide-react-native";
import { useEffect, useRef, useState } from "react";
import { ActivityIndicator, Pressable, RefreshControl, View } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { DayEditSheet } from "@/components/training-day-edit-sheet";
import { ActivityIcon } from "@/components/training-activity-icon";
import { LoadBar } from "@/components/training-dots";
import { HorseSwitcher } from "@/components/training-horse-switcher";
import { Badge, Button, Card, Divider, PageHeader, Screen, Text } from "@/components/ui";
import {
  trainingError,
  useApplyPlan,
  useProfile,
  usePlanWeek,
  useReproposeDay,
  useTakeDay,
  useTrainingHorses,
  useWeek,
  type WeekDay,
} from "@/lib/api/training";
import { useHorse } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";
import { weekRehaText } from "@/lib/reha";
import { colors } from "@/lib/theme";
import { useExerciseList } from "@/lib/tracking-queries";
import { ACTIVITIES, addDays, dayStatusLabel, formatDayLong, weekRangeLabel, weekdayShort } from "@/lib/training";
import {
  applicableDays,
  canAskForAI,
  dayEditBody,
  dayEditFrom,
  editedPlanDay,
  openPlanDays,
  planStatusText,
  replaceDraftDay,
  reproposeOptions,
  type DayEdit,
  type PlanDay,
  type PlanResponse,
} from "@/lib/training-plan";

type PlanMeta = Omit<PlanResponse, "days">;
type Draft = { meta: PlanMeta; days: PlanDay[] };

const AI_CAPTION =
  "KI-Vorschläge sind Vorschläge. Entscheide selbst, ob sie zu deinem Pferd passen; im Zweifel gilt, was Tierarzt oder Trainer sagen.";

/**
 * Week view (JAN-59): who trains on which day, show days, load bar and an assessment. Owners and
 * admins plan the open days as a draft right here ("Woche planen", JAN-89, JAN-100): by the rules,
 * and with the owner's consent by a language model whose proposals the rules check. Single days
 * can be changed in the draft and after applying it.
 */
export default function TrainingWeek() {
  const params = useLocalSearchParams<{ horse?: string; plan?: string }>();
  const horses = useTrainingHorses();
  const [horseId, setHorseId] = useState<string | undefined>(params.horse);
  const [start, setStart] = useState<string>();

  const list = horses.data ?? [];
  const activeId = horseId ?? list[0]?.id;
  const week = useWeek(activeId, start);
  const take = useTakeDay(activeId ?? "");
  const plan = usePlanWeek(activeId ?? "");
  const apply = useApplyPlan(activeId ?? "");
  const consent = useConsentPrompt();
  const data = week.data;

  const [draft, setDraft] = useState<Draft | null>(null);
  /** Meta of the last plan request, also after discarding: tells whether the owner can allow the AI. */
  const [lastPlan, setLastPlan] = useState<PlanMeta | null>(null);
  const [editDate, setEditDate] = useState<string | null>(null);
  const [editorUsed, setEditorUsed] = useState(false);
  const keptDate = useRef("");
  if (editDate) keptDate.current = editDate;

  const planWeek = () => {
    if (!activeId || !data) return;
    const requested = data.start;
    apply.reset();
    plan.mutate(
      { start: requested },
      {
        onSuccess: (res) => {
          if (res.start !== requested || res.horse_id !== activeId) return;
          const { days, ...meta } = res;
          setLastPlan(meta);
          setDraft(days.length > 0 ? { meta, days } : null);
        },
      },
    );
  };

  // ?plan=1 (from the training tab) plans once as soon as the week is there.
  const autoPlanned = useRef(false);
  useEffect(() => {
    if (params.plan === "1" && data?.can_edit && !autoPlanned.current) {
      autoPlanned.current = true;
      planWeek();
    }
  });

  if (horses.isPending || (week.isPending && !!activeId)) {
    return (
      <Screen back>
        <View className="items-center py-16">
          <ActivityIndicator color={colors.primary.DEFAULT} />
        </View>
      </Screen>
    );
  }
  if (!activeId || week.isError || !data) {
    return (
      <Screen back>
        <Card className="gap-3">
          <Text variant="secondary">{week.isError ? trainingError(week.error) : "Kein Pferd ausgewählt."}</Text>
          {week.isError ? <Button label="Erneut versuchen" variant="outline" onPress={() => void week.refetch()} /> : null}
        </Card>
      </Screen>
    );
  }

  const horseName = list.find((h) => h.id === activeId)?.name ?? "";
  const drafting = draft !== null;
  const locked = drafting || plan.isPending;
  const shift = (days: number) => {
    setLastPlan(null);
    setStart(addDays(data.start, days));
  };
  const allowAI = async () => {
    if (await consent.ensure("ai_training")) planWeek();
  };
  const discard = () => {
    apply.reset();
    setDraft(null);
  };
  const openEdit = (date: string) => {
    take.reset();
    setEditorUsed(true);
    setEditDate(date);
  };

  const todayIso = data.days.find((d) => d.is_today)?.date ?? localIso();
  const openDays = openPlanDays(data.days);
  const count = draft ? applicableDays(draft.days) : 0;
  const askAI = lastPlan !== null && canAskForAI({ ...lastPlan, days: [] });
  const draftAI = draft !== null && draft.days.some((d) => d.source === "ai");
  const draftHint = draft ? planStatusText(draft.meta.ai_status, draft.meta.owner_is_me) : null;
  const nothingToPlan = !drafting && lastPlan?.ai_status === "nothing_to_plan";

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl refreshing={week.isRefetching} onRefresh={() => void week.refetch()} tintColor={colors.primary.DEFAULT} />
      }
    >
      <View pointerEvents={locked ? "none" : "auto"} className={locked ? "opacity-50" : undefined}>
        <HorseSwitcher
          horses={list}
          selected={activeId}
          onSelect={(id) => {
            if (locked) return;
            setHorseId(id);
            setStart(undefined);
            setLastPlan(null);
          }}
        />
      </View>

      <PageHeader
        eyebrow={`${horseName} · ${weekRangeLabel(data.start, data.end)}`}
        title={data.assessment}
      >
        {drafting ? <Badge variant="accent" label="Entwurf" /> : null}
        <LoadBar segments={data.segments} />
      </PageHeader>

      <View className="flex-row items-center justify-between">
        <Button variant="outline" size="icon" icon={ChevronLeft} accessibilityLabel="Vorherige Woche" disabled={locked} onPress={() => shift(-7)} />
        <Button
          label="Diese Woche"
          variant="ghost"
          size="sm"
          disabled={locked}
          onPress={() => {
            setLastPlan(null);
            setStart(undefined);
          }}
        />
        <Button variant="outline" size="icon" icon={ChevronRight} accessibilityLabel="Nächste Woche" disabled={locked} onPress={() => shift(7)} />
      </View>
      {drafting ? <Text variant="caption">Erst übernehmen oder verwerfen.</Text> : null}

      {data.can_edit && !drafting ? (
        openDays.length > 0 ? (
          <View className="gap-2">
            <Button label="Woche planen" icon={Sparkles} loading={plan.isPending} onPress={planWeek} />
            {askAI ? <Button label="KI-Vorschläge erlauben" variant="ghost" disabled={plan.isPending} onPress={() => void allowAI()} /> : null}
          </View>
        ) : (
          <Text variant="caption">Alle Tage dieser Woche sind vergeben.</Text>
        )
      ) : null}
      {data.can_edit && nothingToPlan && lastPlan ? <Text variant="secondary">{planStatusText(lastPlan.ai_status, lastPlan.owner_is_me)}</Text> : null}
      {plan.isError ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {trainingError(plan.error)}
        </Text>
      ) : null}

      {draft ? (
        <View className="gap-2">
          <Text variant="secondary">{draftAI ? "Mit KI geplant, von den festen Regeln geprüft." : "Aus den festen Regeln."}</Text>
          {draftHint ? <Text variant="secondary">{draftHint}</Text> : null}
          {askAI ? <Button label="KI-Vorschläge erlauben" variant="ghost" disabled={apply.isPending || plan.isPending} onPress={() => void allowAI()} /> : null}
        </View>
      ) : null}

      <Card padded={false}>
        {data.days.map((d, i) => {
          const draftDay = draft?.days.find((x) => x.date === d.date);
          const editable = data.can_edit && (draftDay !== undefined || canEditStoredDay(d, todayIso));
          return (
            <View key={d.date}>
              {i > 0 ? <Divider /> : null}
              <DayRow
                day={d}
                draftDay={draftDay}
                editable={editable && !apply.isPending}
                busy={take.isPending && take.variables?.day === d.date}
                onTake={() => take.mutate({ day: d.date, status: "planned" })}
                onEdit={() => openEdit(d.date)}
              />
            </View>
          );
        })}
      </Card>
      {take.isError && !editDate ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {trainingError(take.error)}
        </Text>
      ) : null}

      {draft ? (
        <View className="gap-3">
          {count > 0 ? (
            <Button
              label={`Übernehmen (${count} ${count === 1 ? "Tag" : "Tage"})`}
              size="lg"
              fullWidth
              loading={apply.isPending}
              onPress={() => apply.mutate(draft.days, { onSuccess: () => setDraft(null) })}
            />
          ) : (
            <Text variant="secondary">Kein Tag mehr im Entwurf.</Text>
          )}
          {apply.isError ? (
            <Text variant="secondary" tone="danger" accessibilityRole="alert">
              {trainingError(apply.error)}
            </Text>
          ) : null}
          <Button label="Verwerfen" variant="outline" fullWidth disabled={apply.isPending} onPress={discard} />
          {draftAI ? <Text variant="caption">{AI_CAPTION}</Text> : null}
        </View>
      ) : null}

      {data.can_edit && editorUsed ? (
        <DayEditor
          horseId={activeId}
          weekStart={data.start}
          days={data.days}
          draft={draft}
          setDraft={setDraft}
          date={editDate ?? keptDate.current}
          open={editDate !== null}
          onClose={() => setEditDate(null)}
        />
      ) : null}
      {consent.sheet}
    </Screen>
  );
}

/** Stored days the owner can still change: planned, claimed or open, and not in the past. */
function canEditStoredDay(day: WeekDay, today: string): boolean {
  if (day.date < today) return false;
  if (day.status === "done" || day.status === "empty") return false;
  if (day.status === "rest") return day.rest_reason === "planned";
  return day.status === "planned" || day.status === "today" || day.status === "open";
}

function localIso(): string {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/**
 * Loads what the day editor needs (profile, riders, exercises) and wires it to the draft or,
 * for saved days, to the week endpoint. Only mounted for owners and admins.
 */
function DayEditor({
  horseId,
  weekStart,
  days,
  draft,
  setDraft,
  date,
  open,
  onClose,
}: {
  horseId: string;
  weekStart: string;
  days: readonly WeekDay[];
  draft: Draft | null;
  setDraft: (update: (current: Draft | null) => Draft | null) => void;
  date: string;
  open: boolean;
  onClose: () => void;
}) {
  const { user } = useAuth();
  const profile = useProfile(horseId);
  const horse = useHorse(horseId);
  const exercises = useExerciseList();
  const take = useTakeDay(horseId);
  const repropose = useReproposeDay(horseId);
  /** Planned mode: the re-proposed day, shown in the sheet but not saved before "Speichern". */
  const [proposal, setProposal] = useState<DayEdit | null>(null);

  useEffect(() => {
    setProposal(null);
    take.reset();
    repropose.reset();
  }, [open, date]);

  const day = days.find((d) => d.date === date);
  if (!day) return null;
  const draftDay = draft?.days.find((d) => d.date === date);
  const mode = draftDay ? "draft" : "planned";
  const shown = draftDay ?? day;
  const value = proposal ?? dayEditFrom(shown);

  const p = profile.data;
  const fromProfile = p?.exists ? ACTIVITIES.filter((a) => p.allowed_activities.some((x) => x.activity === a && x.mode !== "off")) : [];
  const allowed = fromProfile.length > 0 ? fromProfile : ACTIVITIES;

  const people: { id: string; name: string }[] = [];
  if (user) people.push({ id: user.id, name: "Ich" });
  for (const r of horse.data?.riders ?? []) if (r.user_id !== user?.id) people.push({ id: r.user_id, name: r.name });
  if (shown.user && !people.some((x) => x.id === shown.user!.id)) people.push({ id: shown.user.id, name: shown.user.name });

  const nothingToRelease = day.status === "open" || (day.status === "today" && !day.user && !day.activity);

  const save = (edit: DayEdit) => {
    if (draftDay) {
      if (JSON.stringify(edit) !== JSON.stringify(dayEditFrom(draftDay))) {
        setDraft((d) => (d ? { ...d, days: replaceDraftDay(d.days, editedPlanDay(draftDay, edit)) } : d));
      }
      onClose();
      return;
    }
    take.mutate({ day: date, ...dayEditBody(edit) }, { onSuccess: onClose });
  };
  const remove = () => {
    if (draftDay) {
      setDraft((d) => (d ? { ...d, days: d.days.filter((x) => x.date !== date) } : d));
      onClose();
      return;
    }
    take.mutate({ day: date, status: "open" }, { onSuccess: onClose });
  };
  const proposeAgain = (current: DayEdit) => {
    // Context is what the sheet shows now; the activity shown is the one not to propose again.
    const options =
      draftDay && draft
        ? reproposeOptions(replaceDraftDay(draft.days, editedPlanDay(draftDay, current)), date)
        : { ...reproposeOptions([], date), exclude: current.activity ? [current.activity] : [] };
    repropose.mutate(
      { start: weekStart, repropose: options },
      {
        onSuccess: (res) => {
          const fresh = res.days[0];
          if (!fresh) return;
          if (draftDay) {
            setDraft((d) => {
              if (!d) return d;
              const next = replaceDraftDay(d.days, fresh);
              return {
                meta: { ...d.meta, source: next.some((x) => x.source === "ai") ? "ai" : "rules", ai_status: res.ai_status === "used" ? "used" : d.meta.ai_status },
                days: next,
              };
            });
          } else {
            setProposal({ ...dayEditFrom(fresh), userId: current.userId });
          }
        },
      },
    );
  };

  const error = take.isError ? trainingError(take.error) : repropose.isError ? trainingError(repropose.error) : null;

  return (
    <DayEditSheet
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose();
      }}
      date={date}
      mode={mode}
      value={value}
      allowed={allowed}
      people={people}
      exercises={exercises.data ?? []}
      discipline={p?.discipline ?? ""}
      busy={take.isPending}
      reproposing={repropose.isPending}
      error={error}
      canRemove={mode === "draft" || !nothingToRelease}
      onSave={save}
      onRepropose={proposeAgain}
      onRemove={remove}
    />
  );
}

function DayRow({
  day,
  draftDay,
  editable,
  busy,
  onTake,
  onEdit,
}: {
  day: WeekDay;
  draftDay?: PlanDay;
  editable: boolean;
  busy: boolean;
  onTake: () => void;
  onEdit: () => void;
}) {
  const router = useRouter();
  const label = dayStatusLabel(day.status, day.user, day.is_me, day.rest_reason);
  const open = day.status === "open" || (day.status === "today" && !day.user);
  const summary = draftDay ? draftSummary(draftDay) : label;
  const head = (
    <>
      <View className="w-12 items-center">
        <Text variant="bodyStrong" className={day.is_today ? "text-accent-text" : undefined}>
          {weekdayShort(day.date)}
        </Text>
        <Text variant="caption">{Number(day.date.slice(8, 10))}.</Text>
      </View>
      <View className="flex-1 gap-0.5">
        {draftDay ? (
          <DraftBody day={draftDay} />
        ) : (
          <>
            <Text variant="body" tone={open ? "muted" : "default"}>
              {label}
            </Text>
            {day.activity ? (
              <View className="flex-row items-center gap-1.5">
                <ActivityIcon activity={day.activity} size={14} className="text-muted" />
                <Text variant="caption">
                  {day.label}
                  {day.minutes > 0 ? ` · ${day.minutes} Min.` : ""}
                </Text>
              </View>
            ) : null}
            {day.focus ? <Text variant="caption">Schwerpunkt: {day.focus}</Text> : null}
            {day.exercise ? (
              <Button
                label={`Übung: ${day.exercise.title}`}
                variant="ghost"
                size="sm"
                icon={BookOpen}
                className="self-start px-0"
                onPress={() => router.push(`/training/exercises/${day.exercise!.id}` as Href)}
              />
            ) : null}
            {day.note && day.status !== "done" ? (
              <Text variant="caption" numberOfLines={2}>
                {day.note}
              </Text>
            ) : null}
          </>
        )}
      </View>
    </>
  );
  return (
    <View className="gap-2 p-4" accessible={false}>
      <View className="min-h-touch flex-row items-center gap-3">
        {editable ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`${formatDayLong(day.date)}: ${summary}`}
            accessibilityHint="Tag bearbeiten"
            onPress={onEdit}
            className="flex-1 flex-row items-center gap-3 active:opacity-70"
          >
            {head}
          </Pressable>
        ) : (
          <View
            className="flex-1 flex-row items-center gap-3"
            accessible
            accessibilityLabel={`${formatDayLong(day.date)}: ${summary}`}
          >
            {head}
          </View>
        )}
        {day.can_take && !day.is_me && !draftDay ? (
          <Button label="Ich" size="sm" variant="secondary" loading={busy} onPress={onTake} />
        ) : null}
      </View>
      {day.reha ? (
        <View className="ml-[60px] flex-row items-center gap-2" accessibilityLabel={`Reha: ${weekRehaText(day.reha)}`}>
          <Badge variant={day.reha.done ? "primary" : "info"} label="Reha" />
          <Text variant="secondary" className="flex-1">
            {weekRehaText(day.reha)}
          </Text>
        </View>
      ) : null}
      {day.show ? (
        <View className="ml-[60px] gap-1 rounded-tile bg-accent-soft p-3" accessibilityLabel={`Turnier ${day.show.name}`}>
          <View className="flex-row items-center gap-2">
            <Badge variant="accent" label="Turnier" />
            <Text variant="bodyStrong" className="flex-1">
              {day.show.name}
            </Text>
          </View>
          {day.show.classes ? <Text variant="secondary">Klassen: {day.show.classes}</Text> : null}
          {day.show.helper ? <Text variant="secondary">Helfer: {day.show.helper}</Text> : null}
        </View>
      ) : null}
    </View>
  );
}

function draftSummary(day: PlanDay): string {
  const what = day.activity === "rest" ? day.label : `${day.label}${day.minutes > 0 ? `, ${day.minutes} Min.` : ""}`;
  return `Entwurf: ${what}`;
}

/** The proposal of one day in the draft: badges, unit, who, focus, exercise and the reason. */
function DraftBody({ day }: { day: PlanDay }) {
  const rest = day.activity === "rest";
  return (
    <>
      <View className="flex-row flex-wrap items-center gap-2">
        <Badge variant="neutral" label="Entwurf" />
        {day.edited ? (
          <Badge variant="neutral" label="Geändert" />
        ) : (
          <Badge variant={day.source === "ai" ? "info" : "neutral"} label={day.source === "ai" ? "KI" : "Regel"} />
        )}
      </View>
      <View className="flex-row items-center gap-1.5">
        <ActivityIcon activity={day.activity} size={16} />
        <Text variant="body" className="flex-1">
          {day.label}
          {day.minutes > 0 ? ` · ${day.minutes} Min.` : ""}
        </Text>
      </View>
      {day.level_label && !rest ? <Text variant="caption">{day.level_label}</Text> : null}
      {day.user ? <Text variant="caption">{rest ? `${day.user.name} bleibt eingetragen` : `für ${day.user.name}`}</Text> : null}
      {day.focus ? <Text variant="caption">Schwerpunkt: {day.focus}</Text> : null}
      {day.exercise ? <Text variant="caption">Übung: {day.exercise.title}</Text> : null}
      {!day.edited && day.reason ? <Text variant="secondary">{day.reason}</Text> : null}
      {!day.edited && day.note ? <Text variant="caption">Bedingung: {day.note}</Text> : null}
    </>
  );
}
