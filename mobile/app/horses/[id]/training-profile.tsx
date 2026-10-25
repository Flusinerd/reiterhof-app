import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { Activity, CalendarRange, Medal, Target, Timer, Trophy, Users } from "lucide-react-native";
import { useEffect, useMemo, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { GoalSheet } from "@/components/training-goal-sheet";
import { LevelUpSheet } from "@/components/training-level-up-sheet";
import { progressRows } from "@/components/training-progress";
import { RiderRuleCard } from "@/components/training-rider-rules";
import { Button, Icon, Input, PageHeader, Pill, Screen, Section, Sheet, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { trainingError, useFocuses, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import { dateToIso } from "@/lib/date-field";
import { horseRoutes } from "@/lib/horse-format";
import { ACTIVITIES, DISCIPLINES, PROFILE_STATUSES, activityLabel, disciplineLabel, statusLabel } from "@/lib/training";
import { focusWarnings, focusesSummary } from "@/lib/training-focus";
import { levelUp, levelUpLabel, progressText, type GoalLevel } from "@/lib/training-goal";
import {
  activitySummary,
  draftFromProfile,
  draftToInput,
  rhythmSummary,
  ridersSummary,
  showsSummary,
  structureSummary,
  type ProfileDraft,
} from "@/lib/training-profile";

type EditorSection = "activities" | "rhythm" | "structure" | "shows" | "riders" | "focuses";

const STATUS_TONE = { fit: "success", reha: "accent", pause: "muted" } as const;

/** "Dressur, Niveau L" */
function disciplineLine(d: Pick<ProfileDraft, "discipline" | "level">): string {
  const level = d.level.trim();
  return `${disciplineLabel(d.discipline)}${level ? `, Niveau ${level}` : ""}`;
}

/** The rider's view of the horse's rules: "Erlaubt: Halle, Platz · Bedingt: Ausritt (nur mit Begleitung)". */
function allowedLine(d: Pick<ProfileDraft, "modes">): string {
  const on = ACTIVITIES.filter((a) => d.modes[a].mode === "on").map(activityLabel);
  const conditional = ACTIVITIES.filter((a) => d.modes[a].mode === "conditional").map(
    (a) => `${activityLabel(a)} (${d.modes[a].note})`,
  );
  return [on.length ? `Erlaubt: ${on.join(", ")}` : "", conditional.length ? `Bedingt: ${conditional.join(", ")}` : ""]
    .filter(Boolean)
    .join(" · ");
}

/**
 * Training profile (JAN-55, reworked JAN-99): for the owner (and admins) the status and one row per
 * section with a summary, each opening its own editor; riders see the horse's rules and their own.
 * A horse without a profile goes to the setup wizard (owner) first.
 */
export default function TrainingProfile() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const profile = useProfile(id);
  const save = useSaveProfile(id ?? "");
  const [error, setError] = useState<string | null>(null);
  // Discipline and level are edited in a sheet; it keeps its own copy while open.
  const [sheetOpen, setSheetOpen] = useState(false);
  const [discipline, setDiscipline] = useState("");
  const [level, setLevel] = useState("");
  // The level goal and the one-tap level change have their own sheets.
  const [goalOpen, setGoalOpen] = useState(false);
  const [levelUpOpen, setLevelUpOpen] = useState(false);
  const catalog = useFocuses();

  const data = profile.data;
  const draft = useMemo(() => (data?.exists ? draftFromProfile(data) : null), [data]);
  const toSetup = !!data && !data.exists && data.can_edit;
  useEffect(() => {
    if (toSetup && id) router.replace(horseRoutes.trainingSetup(id) as Href);
  }, [toSetup, id, router]);

  if (profile.isPending || toSetup) {
    return (
      <Screen back>
        <View className="items-center py-16">
          <ActivityIndicator />
        </View>
      </Screen>
    );
  }
  if (profile.isError || !data) {
    return (
      <Screen back>
        <View className="gap-3">
          <Text variant="secondary">{profile.isError ? trainingError(profile.error) : "Kein Profil gefunden."}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void profile.refetch()} />
        </View>
      </Screen>
    );
  }

  const editable = data.can_edit;
  const name = data.horse_name;
  const go = (section: EditorSection) => router.push(horseRoutes.trainingProfileSection(id!, section) as Href);

  /** Stores the profile with one change; the draft comes from the cache, so nothing else is touched. */
  const saveChange = (change: Partial<ProfileDraft>, onSuccess?: () => void) => {
    if (!draft) return;
    const result = draftToInput({ ...draft, ...change });
    if (!result.ok) {
      setError(result.error);
      return;
    }
    setError(null);
    save.mutate(result.input as ProfileInput, { onSuccess });
  };

  const openSheet = () => {
    if (!draft) return;
    setDiscipline(draft.discipline);
    setLevel(draft.level);
    setError(null);
    save.reset();
    setSheetOpen(true);
  };

  const progress = data.progress ?? null;
  const levelUpInfo = levelUp(data.warnings, progress);
  const openExercise = (exerciseId: string) => router.push(`/training/exercises/${exerciseId}` as Href);
  const openSheetFor = (open: (v: boolean) => void) => () => {
    setError(null);
    save.reset();
    open(true);
  };
  /** Sets the level goal; only this save carries it, so that no other editor touches it. */
  const saveGoal = (goal: { level: GoalLevel | null; until: string }) => saveChange({ goal }, () => setGoalOpen(false));
  /** "Auf L einstufen": the new level, the goal back to the class above it. */
  const confirmLevelUp = () => {
    if (levelUpInfo?.letter) saveChange({ level: levelUpInfo.letter, goal: { level: null, until: "" } }, () => setLevelUpOpen(false));
  };

  const shownError = error ?? (save.isError ? trainingError(save.error) : null);

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <PageHeader
        eyebrow={name}
        title="Trainingsprofil"
        description={
          editable
            ? "Bestimmt die Empfehlungen und die Wochenplanung."
            : "Nur der Besitzer kann es ändern. Du siehst, was für dich gilt."
        }
      >
        {draft ? (
          <Text variant="eyebrow" tone={STATUS_TONE[draft.status]}>
            {statusLabel(draft.status)}
          </Text>
        ) : (
          <Text variant="eyebrow" tone="accent">
            Noch nicht angelegt
          </Text>
        )}
      </PageHeader>

      {!draft ? (
        <Text variant="secondary">Der Besitzer hat noch kein Profil angelegt.</Text>
      ) : editable ? (
        <>
          <Section title="Status">
            <View className="flex-row flex-wrap gap-2">
              {PROFILE_STATUSES.map((s) => (
                <Pill
                  key={s.value}
                  label={s.label}
                  selected={draft.status === s.value}
                  disabled={save.isPending}
                  onPress={() => {
                    if (s.value !== draft.status) saveChange({ status: s.value });
                  }}
                />
              ))}
            </View>
            {shownError && !sheetOpen ? (
              <Text variant="secondary" tone="danger" accessibilityRole="alert">
                {shownError}
              </Text>
            ) : null}
          </Section>

          <RowList>
            <ListRow
              leading={<Icon as={Medal} size={20} className="text-muted" />}
              title="Disziplin und Niveau"
              detail={disciplineLine(draft)}
              accessibilityLabel="Disziplin und Niveau"
              onPress={openSheet}
            />
            {progress
              ? progressRows({
                  text: progressText(progress, name),
                  next: progress.next,
                  onOpenGoal: openSheetFor(setGoalOpen),
                  onOpenExercise: openExercise,
                })
              : null}
            {levelUpInfo ? (
              <ListRow
                key="level-up"
                className="border-l-2 border-accent pl-3"
                eyebrow="Hinweis"
                eyebrowTone="accent"
                title={levelUpInfo.text}
              >
                {levelUpInfo.letter ? (
                  <Button
                    label={levelUpLabel(levelUpInfo.letter)}
                    variant="outline"
                    size="sm"
                    className="mt-2"
                    onPress={openSheetFor(setLevelUpOpen)}
                  />
                ) : null}
              </ListRow>
            ) : null}
            {draft.focuses !== undefined ? (
              <ListRow
                key="focuses"
                leading={<Icon as={Target} size={20} className="text-muted" />}
                title="Schwerpunkte"
                detail={focusesSummary(data.focuses, catalog.data)}
                accessibilityLabel={`Schwerpunkte: ${focusesSummary(data.focuses, catalog.data)}`}
                onPress={() => go("focuses")}
              >
                {focusWarnings(data.warnings).map((w) => (
                  <Text key={w.key} variant="secondary" tone="accent">
                    {w.text}
                  </Text>
                ))}
              </ListRow>
            ) : null}
            <ListRow
              leading={<Icon as={Activity} size={20} className="text-muted" />}
              title="Aktivitäten"
              detail={activitySummary(draft)}
              accessibilityLabel="Aktivitäten"
              onPress={() => go("activities")}
            />
            <ListRow
              leading={<Icon as={Timer} size={20} className="text-muted" />}
              title="Rhythmus"
              detail={rhythmSummary(draft)}
              accessibilityLabel="Rhythmus"
              onPress={() => go("rhythm")}
            />
            <ListRow
              leading={<Icon as={CalendarRange} size={20} className="text-muted" />}
              title="Feste Tage und Ziele"
              detail={structureSummary(draft)}
              accessibilityLabel="Feste Tage und Ziele"
              onPress={() => go("structure")}
            />
            <ListRow
              leading={<Icon as={Trophy} size={20} className="text-muted" />}
              title="Turniere"
              detail={showsSummary(draft.shows, dateToIso(new Date()))}
              accessibilityLabel="Turniere"
              onPress={() => go("shows")}
            />
            {draft.riders.length > 0 ? (
              <ListRow
                leading={<Icon as={Users} size={20} className="text-muted" />}
                title="Reitbeteiligungen"
                detail={ridersSummary(draft.riders)}
                accessibilityLabel="Reitbeteiligungen"
                onPress={() => go("riders")}
              />
            ) : null}
          </RowList>

          <Sheet
            open={sheetOpen}
            onOpenChange={setSheetOpen}
            title="Disziplin und Niveau"
            description={`Was macht ${name} hauptsächlich?`}
          >
            <View className="flex-row flex-wrap gap-2">
              {DISCIPLINES.map((d) => (
                <Pill key={d.value} label={d.label} selected={discipline === d.value} onPress={() => setDiscipline(d.value)} />
              ))}
            </View>
            <View className="gap-1.5">
              <Text variant="eyebrow">Niveau</Text>
              <Input
                accessibilityLabel="Niveau"
                placeholder="Niveau, z. B. L (optional)"
                value={level}
                maxLength={40}
                onChangeText={setLevel}
              />
            </View>
            {shownError ? (
              <Text variant="secondary" tone="danger" accessibilityRole="alert">
                {shownError}
              </Text>
            ) : null}
            <Button
              label="Speichern"
              size="lg"
              fullWidth
              loading={save.isPending}
              onPress={() => saveChange({ discipline, level }, () => setSheetOpen(false))}
            />
          </Sheet>

          <GoalSheet
            open={goalOpen}
            onOpenChange={setGoalOpen}
            horseName={name}
            progress={progress}
            saving={save.isPending}
            error={shownError}
            onSave={saveGoal}
          />
          {levelUpInfo?.letter ? (
            <LevelUpSheet
              open={levelUpOpen}
              onOpenChange={setLevelUpOpen}
              horseName={name}
              letter={levelUpInfo.letter}
              saving={save.isPending}
              error={shownError}
              onConfirm={confirmLevelUp}
            />
          ) : null}
        </>
      ) : (
        <>
          <Text variant="body">
            {[disciplineLine(draft), allowedLine(draft)].filter(Boolean).join(" · ")}
          </Text>
          {data.focuses && data.focuses.length > 0 ? (
            <Text variant="body">{`Schwerpunkte: ${focusesSummary(data.focuses, catalog.data)}`}</Text>
          ) : null}
          {progress ? (
            <RowList>{progressRows({ text: progressText(progress, name), next: progress.next, onOpenExercise: openExercise })}</RowList>
          ) : null}
          {draft.riders.length > 0 ? (
            <Section title="Meine Regeln">
              {draft.riders.map((r) => (
                <RiderRuleCard key={r.userId} rider={r} editable={false} />
              ))}
            </Section>
          ) : null}
        </>
      )}
    </Screen>
  );
}
