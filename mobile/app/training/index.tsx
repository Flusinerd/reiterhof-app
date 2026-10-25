import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { BookOpen, Check, HeartPulse, ListChecks, Pencil, Play, SlidersHorizontal } from "lucide-react-native";
import { useMemo, useState } from "react";
import { ActivityIndicator, RefreshControl, ScrollView, View } from "react-native";

import { HorseWeekStrip } from "@/components/horse-week-strip";
import { ActivityIcon } from "@/components/training-activity-icon";
import { ActivityPicker } from "@/components/training-activity-picker";
import { HorseSwitcher } from "@/components/training-horse-switcher";
import { NextStepCard } from "@/components/training-next-step";
import { QuickLogSheet } from "@/components/training-quick-log";
import { StepsSheet } from "@/components/training-steps-sheet";
import { Button, Icon, PageHeader, Pill, Screen, Section, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import {
  trainingError,
  useToday,
  useTrainingHorses,
  useWeek,
  type Recommendation,
} from "@/lib/api/training";
import { horseRoutes } from "@/lib/horse-format";
import { horseWeekStrip } from "@/lib/horse-page";
import { savedTitle } from "@/lib/session-finish";
import { useColors } from "@/lib/theme";
import {
  DEFAULT_MINUTES,
  TIME_OPTIONS,
  activityLabel,
  formatMinutes,
  intensityLabel,
  isActivity,
  statusLabel,
  trainedCount,
  type Activity,
} from "@/lib/training";
import { nextStep } from "@/lib/training-plan";

/**
 * "Was heute?" (JAN-57): the horse, the recommendation for today as the one raised block with its
 * primary action, alternatives as rows, the week as one strip, and the links to profile and library.
 */
export default function Training() {
  const router = useRouter();
  const c = useColors();
  const params = useLocalSearchParams<{ horse?: string }>();
  const horses = useTrainingHorses();
  const [horseId, setHorseId] = useState<string | undefined>(params.horse);
  const [minutes, setMinutes] = useState(45);
  const [choice, setChoice] = useState(0);
  const [picked, setPicked] = useState<Activity | null>(null);
  const [stepsOpen, setStepsOpen] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [logOpen, setLogOpen] = useState(false);
  // What "Nur eintragen" stored on this visit ("Halle, 45 Min."): the sheet closes, so the page says so.
  const [logged, setLogged] = useState<string | null>(null);

  const list = horses.data ?? [];
  const activeId = list.some((h) => h.id === horseId) ? horseId : list[0]?.id;
  const today = useToday(activeId, minutes);
  const data = today.data;
  const week = useWeek(activeId);
  const step = nextStep(data, week.data);
  const strip = useMemo(() => {
    const days = week.data?.days;
    if (!days) return [];
    return horseWeekStrip(days, days.find((d) => d.is_today)?.date ?? "");
  }, [week.data]);

  const selectHorse = (id: string) => {
    setHorseId(id);
    setChoice(0);
    setPicked(null);
    setLogged(null);
  };

  if (horses.isPending) {
    return (
      <Screen back>
        <PageHeader title="Training" />
        <ActivityIndicator color={c.primary.DEFAULT} className="self-start" />
      </Screen>
    );
  }
  if (horses.isError) {
    return (
      <Screen back>
        <PageHeader title="Training" />
        <View className="gap-3">
          <Text variant="bodyStrong">Laden fehlgeschlagen</Text>
          <Text variant="secondary">{trainingError(horses.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void horses.refetch()} />
        </View>
      </Screen>
    );
  }
  if (list.length === 0 || !activeId) {
    return (
      <Screen back>
        <PageHeader
          eyebrow="Training"
          title="Noch kein Pferd"
          description="Die Empfehlung erscheint, sobald du Besitzer oder Reitbeteiligung bist."
        />
      </Screen>
    );
  }

  const horse = list.find((h) => h.id === activeId)!;
  const recs = data?.recommendations ?? [];
  const hiddenActivities = (data?.hidden ?? []).map((h) => h.activity);
  const byActivity = (a: string) => recs.find((r) => r.activity === a);
  const chosenRec: Recommendation | undefined = picked ? byActivity(picked) : recs[choice];
  const own: Recommendation | undefined = picked
    ? {
        activity: picked,
        label: activityLabel(picked),
        minutes: DEFAULT_MINUTES[picked],
        intensity: "none",
        intensity_label: "",
        reason: "Deine Wahl.",
      }
    : undefined;
  const hero: Recommendation | undefined = chosenRec ?? own;
  const alternatives = recs.filter((r) => r.activity !== hero?.activity).slice(0, 2);
  const isRest = hero?.activity === "rest";
  // Without a profile the setup is the one thing to do; "Starten" steps back to an outline.
  const setupFirst = step?.kind === "profile";

  const start = () => {
    if (!hero || !isActivity(hero.activity)) return;
    router.push({
      pathname: "/training/session",
      params: {
        horse: activeId,
        activity: hero.activity,
        minutes: String(hero.minutes),
        ...(hero.exercise ? { exercise: hero.exercise.id } : {}),
      },
    });
  };

  const goto = (href: string) => router.push(href as Href);
  const weekHref = `/training/week?horse=${activeId}`;

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl refreshing={today.isRefetching} onRefresh={() => void today.refetch()} tintColor={c.primary.DEFAULT} />
      }
    >
      <PageHeader
        eyebrow={data ? `Training · ${statusLabel(data.status)}` : "Training"}
        title={horse.name}
        description={data?.context}
      >
        <HorseSwitcher horses={list} selected={activeId} onSelect={selectHorse} />
      </PageHeader>

      {logged ? (
        <View className="flex-row items-center gap-2">
          <Icon as={Check} size={18} className="text-success" />
          <Text variant="bodyStrong" tone="success">{`Eingetragen: ${logged}`}</Text>
        </View>
      ) : null}

      {!data && !today.isError ? <ActivityIndicator color={c.primary.DEFAULT} className="self-start" /> : null}

      {today.isError ? (
        <View className="gap-3">
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {trainingError(today.error)}
          </Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void today.refetch()} />
        </View>
      ) : null}

      {data && !data.has_profile && !data.can_edit ? (
        <Text variant="secondary">Ohne Profil keine Empfehlung. Der Besitzer muss es anlegen.</Text>
      ) : null}

      {setupFirst ? <NextStepCard step={step} horseId={activeId} horseName={horse.name} /> : null}

      {hero ? (
        <View className="gap-4 rounded-card border border-border bg-raised p-5">
          <View className="flex-row items-center gap-4">
            {!isRest ? (
              <View className="h-14 w-14 items-center justify-center rounded-pill bg-primary-soft">
                <ActivityIcon activity={hero.activity} size={26} />
              </View>
            ) : null}
            <View className="flex-1 gap-1">
              <Text variant="eyebrow" tone="brass">
                {isRest ? "Empfehlung" : `Empfehlung · ${formatMinutes(hero.minutes)}`}
              </Text>
              <Text variant="titleLg" accessibilityRole="header">
                {hero.label}
              </Text>
            </View>
          </View>
          <View className="gap-1">
            <Text variant="bodySm" tone="muted">
              {hero.reason}
            </Text>
            {hero.note ? (
              <Text variant="secondary" tone="accent">
                Bedingung: {hero.note}
              </Text>
            ) : null}
            {!isRest ? (
              <Text variant="secondary">
                {hero.intensity_label ? `Belastung ${hero.intensity_label}` : intensityLabel("none")}
              </Text>
            ) : null}
          </View>
          {!isRest ? (
            <View className="gap-2">
              <Button
                label="Starten"
                icon={Play}
                fullWidth
                variant={setupFirst ? "outline" : "primary"}
                onPress={start}
                disabled={!data?.can_log}
              />
              {hero.exercise ? (
                <Button label="Ablauf ansehen" variant="ghost" size="sm" fullWidth onPress={() => setStepsOpen(true)} />
              ) : null}
            </View>
          ) : null}
        </View>
      ) : null}

      {!setupFirst && step ? <NextStepCard step={step} horseId={activeId} horseName={horse.name} /> : null}

      <View className="gap-2">
        <Text variant="eyebrow">Zeit heute</Text>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-2">
          {TIME_OPTIONS.map((m) => (
            <Pill
              key={m}
              label={`${m} Min.`}
              selected={minutes === m}
              onPress={() => {
                setMinutes(m);
                setChoice(0);
                setPicked(null);
              }}
            />
          ))}
        </ScrollView>
        <View className="-ml-4 flex-row flex-wrap">
          <Button label="Andere Aktivität" variant="ghost" size="sm" icon={ListChecks} onPress={() => setPickerOpen(true)} />
          {data?.can_log ? (
            <Button label="Nur eintragen" variant="ghost" size="sm" icon={Pencil} onPress={() => setLogOpen(true)} />
          ) : null}
        </View>
      </View>

      {alternatives.length > 0 ? (
        <Section title="Oder lieber">
          <RowList>
            {alternatives.map((r) => (
              <ListRow
                key={r.activity}
                leading={<ActivityIcon activity={r.activity} size={22} className="text-muted" />}
                eyebrow={r.activity === "rest" ? undefined : formatMinutes(r.minutes)}
                eyebrowTone="brass"
                title={r.label}
                detail={r.reason}
                accessibilityLabel={`${r.label}, ${r.activity === "rest" ? "" : formatMinutes(r.minutes) + ", "}${r.reason}`}
                onPress={() => {
                  setPicked(null);
                  setChoice(recs.findIndex((x) => x.activity === r.activity));
                }}
              />
            ))}
          </RowList>
        </Section>
      ) : null}

      {data && data.hidden.length > 0 ? (
        <View className="gap-1">
          {data.hidden.map((h) => (
            <Text key={h.activity} variant="caption">
              {h.reason}
            </Text>
          ))}
        </View>
      ) : null}

      <Section
        title="Diese Woche"
        description={data ? `${trainedCount(data.week)} von 7 Tagen trainiert` : undefined}
        action={<Button label="Woche" variant="ghost" size="sm" onPress={() => goto(weekHref)} />}
      >
        {week.isPending ? (
          <ActivityIndicator color={c.primary.DEFAULT} className="self-start" />
        ) : strip.length > 0 ? (
          <HorseWeekStrip items={strip} onPress={() => goto(weekHref)} />
        ) : null}
      </Section>

      <RowList>
        <ListRow
          leading={<Icon as={BookOpen} size={20} className="text-muted" />}
          title="Übungen"
          accessibilityLabel="Übungen"
          onPress={() => goto("/training/exercises")}
        />
        <ListRow
          leading={<Icon as={SlidersHorizontal} size={20} className="text-muted" />}
          title="Trainingsprofil"
          accessibilityLabel="Trainingsprofil"
          onPress={() => goto(horseRoutes.trainingProfile(activeId))}
        />
        {data?.reha || data?.status === "reha" ? (
          <ListRow
            leading={<Icon as={HeartPulse} size={20} className="text-muted" />}
            title="Reha-Plan"
            accessibilityLabel="Reha-Plan"
            onPress={() => goto(horseRoutes.reha(activeId))}
          />
        ) : null}
      </RowList>

      <StepsSheet exercise={hero?.exercise} open={stepsOpen} onOpenChange={setStepsOpen} />
      <ActivityPicker
        hidden={hiddenActivities}
        selected={hero?.activity}
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onPick={setPicked}
      />
      <QuickLogSheet
        horseId={activeId}
        horseName={horse.name}
        hidden={hiddenActivities}
        open={logOpen}
        onOpenChange={setLogOpen}
        onSaved={({ activity, minutes }) => setLogged(savedTitle(activity, minutes))}
      />
    </Screen>
  );
}
