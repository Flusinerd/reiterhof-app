import { useRouter, type Href } from "expo-router";
import { BookOpen, CalendarDays, ChevronRight, ListChecks, Pencil, Play } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, RefreshControl, View } from "react-native";

import { ActivityPicker } from "@/components/training-activity-picker";
import { ActivityIcon } from "@/components/training-activity-icon";
import { WeekDots } from "@/components/training-dots";
import { HorseSwitcher } from "@/components/training-horse-switcher";
import { QuickLogSheet } from "@/components/training-quick-log";
import { StepsSheet } from "@/components/training-steps-sheet";
import {
  Badge,
  Button,
  Card,
  Icon,
  PageHeader,
  Pill,
  PressableCard,
  Screen,
  Section,
  Text,
} from "@/components/ui";
import {
  trainingError,
  useToday,
  useTrainingHorses,
  type Recommendation,
} from "@/lib/api/training";
import { colors } from "@/lib/theme";
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

/** "Was heute?" (JAN-57): the recommendation for today, 7-day status, alternatives, quick log. */
export default function Training() {
  const router = useRouter();
  const horses = useTrainingHorses();
  const [horseId, setHorseId] = useState<string>();
  const [minutes, setMinutes] = useState(45);
  const [choice, setChoice] = useState(0);
  const [picked, setPicked] = useState<Activity | null>(null);
  const [stepsOpen, setStepsOpen] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [logOpen, setLogOpen] = useState(false);

  const list = horses.data ?? [];
  const activeId = list.some((h) => h.id === horseId) ? horseId : list[0]?.id;
  const today = useToday(activeId, minutes);
  const data = today.data;

  const selectHorse = (id: string) => {
    setHorseId(id);
    setChoice(0);
    setPicked(null);
  };

  if (horses.isPending) {
    return (
      <Screen>
        <PageHeader title="Training" />
        <ActivityIndicator color={colors.primary.DEFAULT} className="self-start" />
      </Screen>
    );
  }
  if (horses.isError) {
    return (
      <Screen>
        <PageHeader title="Training" />
        <Card className="gap-3">
          <Text variant="bodyStrong">Laden fehlgeschlagen</Text>
          <Text variant="secondary">{trainingError(horses.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void horses.refetch()} />
        </Card>
      </Screen>
    );
  }
  if (list.length === 0 || !activeId) {
    return (
      <Screen>
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

  return (
    <Screen
      refreshControl={
        <RefreshControl refreshing={today.isRefetching} onRefresh={() => void today.refetch()} tintColor={colors.primary.DEFAULT} />
      }
    >
      <HorseSwitcher horses={list} selected={activeId} onSelect={selectHorse} />

      {/* Head: horse, profile status, the last 7 days, context */}
      <PageHeader eyebrow="Training" title={horse.name} description={data?.context}>
        {data ? (
          <View className="gap-3">
            <WeekDots dots={data.week} />
            <Badge
              variant={data.status === "fit" ? "primary" : "accent"}
              label={`${statusLabel(data.status)} · ${trainedCount(data.week)}/7 Tage trainiert`}
            />
          </View>
        ) : (
          <ActivityIndicator color={colors.primary.DEFAULT} className="self-start" />
        )}
      </PageHeader>

      {today.isError ? (
        <Card className="gap-3">
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {trainingError(today.error)}
          </Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void today.refetch()} />
        </Card>
      ) : null}

      {data && !data.has_profile ? (
        <Card className="gap-3">
          <Text variant="bodyStrong">Noch kein Trainingsprofil</Text>
          <Text variant="secondary">
            {data.can_edit
              ? `Lege fest, was für ${data.horse_name} in Frage kommt.`
              : `Ohne Profil keine Empfehlung. Der Besitzer muss es anlegen.`}
          </Text>
          {data.can_edit ? (
            <Button label="Profil anlegen" onPress={() => goto(`/horses/${activeId}/training-profile`)} />
          ) : null}
        </Card>
      ) : null}

      {hero ? (
        <Card className="gap-3">
          <Text variant="label">{isRest ? "Empfehlung" : `Empfehlung · ${formatMinutes(hero.minutes)}`}</Text>
          <View className="flex-row items-center gap-3">
            {!isRest ? <ActivityIcon activity={hero.activity} size={28} /> : null}
            <Text variant="title" accessibilityRole="header" className="flex-1">
              {hero.label}
            </Text>
          </View>
          <Text variant="body" tone="muted">
            {hero.reason}
          </Text>
          {hero.note ? (
            <Text variant="secondary" tone="accent">
              Bedingung: {hero.note}
            </Text>
          ) : null}
          {!isRest ? (
            <>
              <Text variant="secondary">
                {hero.intensity_label ? `Belastung ${hero.intensity_label}` : intensityLabel("none")}
              </Text>
              <View className="mt-1 flex-row flex-wrap gap-2">
                <Button label="Starten" icon={Play} onPress={start} disabled={!data?.can_log} />
                {hero.exercise ? (
                  <Button label="Ablauf" variant="outline" onPress={() => setStepsOpen(true)} />
                ) : null}
              </View>
            </>
          ) : null}
        </Card>
      ) : null}

      {alternatives.length > 0 ? (
        <Section title="Alternativen">
          {alternatives.map((r) => (
            <PressableCard
              key={r.activity}
              shape="tile"
              className="flex-row items-center gap-3"
              accessibilityLabel={`${r.label}, ${r.activity === "rest" ? "" : formatMinutes(r.minutes) + ", "}${r.reason}`}
              onPress={() => {
                setPicked(null);
                setChoice(recs.findIndex((x) => x.activity === r.activity));
              }}
            >
              <ActivityIcon activity={r.activity} />
              <View className="flex-1 gap-0.5">
                <Text variant="bodyStrong">
                  {r.label}
                  {r.activity === "rest" ? "" : ` · ${formatMinutes(r.minutes)}`}
                </Text>
                <Text variant="secondary" numberOfLines={2}>
                  {r.reason}
                </Text>
              </View>
              <Icon as={ChevronRight} size={20} className="text-muted" />
            </PressableCard>
          ))}
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

      <Section title="Zeit heute">
        <View className="flex-row flex-wrap gap-2">
          {TIME_OPTIONS.map((m) => (
            <Pill key={m} label={`${m} Min.`} selected={minutes === m} onPress={() => { setMinutes(m); setChoice(0); setPicked(null); }} />
          ))}
        </View>
        <View className="flex-row flex-wrap gap-2">
          <Button label="Wählen" variant="outline" icon={ListChecks} onPress={() => setPickerOpen(true)} />
          {data?.can_log ? (
            <Button label="Nur eintragen" variant="outline" icon={Pencil} onPress={() => setLogOpen(true)} />
          ) : null}
        </View>
      </Section>

      <View className="flex-row flex-wrap gap-2">
        <Button
          label="Woche"
          variant="ghost"
          size="sm"
          icon={CalendarDays}
          onPress={() => goto(`/training/week?horse=${activeId}`)}
        />
        <Button label="Übungen" variant="ghost" size="sm" icon={BookOpen} onPress={() => goto("/training/exercises")} />
        <Button label="Profil" variant="ghost" size="sm" onPress={() => goto(`/horses/${activeId}/training-profile`)} />
        {data?.reha || data?.status === "reha" ? (
          <Button label="Reha-Plan" variant="ghost" size="sm" onPress={() => goto(`/horses/${activeId}/reha`)} />
        ) : null}
      </View>

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
      />
    </Screen>
  );
}
