import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { useEffect, useRef, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { ActivityModeList } from "@/components/training-activity-modes";
import { WeekStructureRows } from "@/components/training-week-structure";
import { Button, Card, Divider, Input, PageHeader, Pill, RangeStepper, Screen, Stepper, Switch, Text } from "@/components/ui";
import { trainingError, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import { horseRoutes } from "@/lib/horse-format";
import { DISCIPLINES, PROFILE_STATUSES, disciplineLabel, statusLabel } from "@/lib/training";
import {
  MAX_MINUTES_OPTIONS,
  activitySummary,
  applyDisciplineDefaults,
  draftToInput,
  rhythmSummary,
  setupDraft,
  setupStepError,
  structureSummary,
  type ProfileDraft,
} from "@/lib/training-profile";

type Step = 0 | 1 | 2 | 3 | 4;

const STEP_TITLES = ["Disziplin", "Aktivitäten", "Rhythmus", "Feste Tage", "Zusammenfassung"] as const;

/** One sentence under the title of each step. */
function stepDescription(step: Step, name: string): string {
  switch (step) {
    case 0:
      return `Was macht ${name} hauptsächlich?`;
    case 1:
      return `Was ist für ${name} in Ordnung? Tippen zum Ein- oder Ausschalten.`;
    case 2:
      return "Wie viel soll es pro Woche sein?";
    case 3:
      return "Gibt es feste Regeln je Wochentag? Kannst du überspringen.";
    case 4:
      return `So bekommt ${name} Empfehlungen.`;
  }
}

/** The wizard only offers the common limits; 120 minutes stays available in the rhythm editor. */
const WIZARD_MAX_MINUTES = MAX_MINUTES_OPTIONS.filter((o) => o.value !== 120);

/**
 * Guided setup of the training profile (JAN-98), shown while a horse has no profile: discipline,
 * activities, rhythm, fixed weekdays and a summary. Nothing is stored before the last step.
 */
export default function TrainingSetup() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const profile = useProfile(id);
  const save = useSaveProfile(id ?? "");
  const [step, setStep] = useState<Step>(0);
  const [draft, setDraft] = useState<ProfileDraft>(() => setupDraft());
  const [saveError, setSaveError] = useState<string | null>(null);
  // Set once the profile is being saved, so that the guard below does not send us to the overview
  // when the saved profile arrives in the cache, before the redirect to the training tab.
  const saving = useRef(false);

  const data = profile.data;
  const redirect = !!data && (data.exists || !data.can_edit);
  useEffect(() => {
    if (redirect && id && !saving.current) router.replace(horseRoutes.trainingProfile(id) as Href);
  }, [redirect, id, router]);

  if (profile.isError) {
    return (
      <Screen back>
        <Card className="gap-3">
          <Text variant="secondary">{trainingError(profile.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void profile.refetch()} />
        </Card>
      </Screen>
    );
  }
  if (!data || redirect) {
    return (
      <Screen back>
        <View className="items-center py-16">
          <ActivityIndicator />
        </View>
      </Screen>
    );
  }

  const name = data.horse_name;
  const title = STEP_TITLES[step];
  const patch = (p: Partial<ProfileDraft>) => setDraft((d) => ({ ...d, ...p }));
  const stepError = setupStepError(draft, step);
  const error = stepError ?? saveError ?? (save.isError ? trainingError(save.error) : null);

  const pickDiscipline = (discipline: string) => {
    if (discipline === draft.discipline) return;
    setDraft(applyDisciplineDefaults(draft, discipline));
  };

  const num = (text: string) => Number(text);
  // Sessions and rest days must fit into one week: the other range gives way.
  const setSessions = (min: number, max: number) =>
    patch({
      sessionsMin: String(min),
      sessionsMax: String(max),
      restDaysMin: String(Math.min(num(draft.restDaysMin), 7 - min)),
    });
  const setRestDays = (min: number, max: number) => {
    const sessionsMin = Math.min(num(draft.sessionsMin), 7 - min);
    patch({
      restDaysMin: String(min),
      restDaysMax: String(max),
      sessionsMin: String(sessionsMin),
      sessionsMax: String(Math.max(num(draft.sessionsMax), sessionsMin)),
    });
  };

  const goTo = (next: Step) => {
    setSaveError(null);
    setStep(next);
  };

  const submit = () => {
    const result = draftToInput(draft);
    if (!result.ok) {
      setSaveError(result.error);
      return;
    }
    setSaveError(null);
    saving.current = true;
    save.mutate(result.input as ProfileInput, {
      onSuccess: () => router.replace({ pathname: "/(tabs)/training", params: { horse: id } }),
      onError: () => {
        saving.current = false;
      },
    });
  };

  return (
    // The key remounts the screen per step, which puts the scroll position back to the top.
    <Screen key={step} back keyboardShouldPersistTaps="handled">
      <PageHeader eyebrow={name} title={title} description={stepDescription(step, name)}>
        <Stepper steps={5} current={step} label={title} />
      </PageHeader>

      {step === 0 ? (
        <>
          <View className="gap-3">
            <View className="flex-row flex-wrap gap-2">
              {DISCIPLINES.map((d) => (
                <Pill
                  key={d.value}
                  label={d.label}
                  selected={draft.discipline === d.value}
                  onPress={() => pickDiscipline(d.value)}
                />
              ))}
            </View>
            {draft.discipline ? (
              <Text variant="secondary">
                Vorgaben für {disciplineLabel(draft.discipline)} übernommen. Anpassen im nächsten Schritt.
              </Text>
            ) : null}
          </View>
          <View className="gap-1.5">
            <Text variant="label">Niveau</Text>
            <Input
              accessibilityLabel="Niveau"
              placeholder="Niveau, z. B. L (optional)"
              value={draft.level}
              maxLength={40}
              onChangeText={(level) => patch({ level })}
            />
          </View>
          <View className="gap-1.5">
            <Text variant="label">Status</Text>
            <View className="flex-row flex-wrap gap-2">
              {PROFILE_STATUSES.map((s) => (
                <Pill
                  key={s.value}
                  label={s.label}
                  selected={draft.status === s.value}
                  onPress={() => patch({ status: s.value })}
                />
              ))}
            </View>
          </View>
        </>
      ) : null}

      {step === 1 ? (
        <ActivityModeList
          modes={draft.modes}
          editable
          onChange={(activity, value) => patch({ modes: { ...draft.modes, [activity]: value } })}
        />
      ) : null}

      {step === 2 ? (
        <>
          <RangeStepper
            label="Einheiten pro Woche"
            min={num(draft.sessionsMin)}
            max={num(draft.sessionsMax)}
            lowerBound={1}
            upperBound={7}
            onChange={setSessions}
          />
          <RangeStepper
            label="Ruhetage pro Woche"
            min={num(draft.restDaysMin)}
            max={num(draft.restDaysMax)}
            lowerBound={0}
            upperBound={6}
            onChange={setRestDays}
          />
          <View className="gap-2">
            <Text variant="label">Höchstdauer pro Einheit</Text>
            <View className="flex-row flex-wrap gap-2">
              {WIZARD_MAX_MINUTES.map((o) => (
                <Pill
                  key={o.value}
                  label={o.label}
                  selected={draft.maxMinutes === String(o.value)}
                  onPress={() => patch({ maxMinutes: String(o.value) })}
                />
              ))}
            </View>
          </View>
          <Switch
            label="Ruhetag nach dem Turnier"
            value={draft.restAfterShow}
            onValueChange={(restAfterShow) => patch({ restAfterShow })}
          />
        </>
      ) : null}

      {step === 3 ? (
        <WeekStructureRows days={draft.days} modes={draft.modes} editable onChange={(days) => patch({ days })} />
      ) : null}

      {step === 4 ? (
        <>
          <Card padded={false}>
            <SummaryRow
              label="Disziplin"
              value={[disciplineLabel(draft.discipline), draft.level.trim() ? `Niveau ${draft.level.trim()}` : "", statusLabel(draft.status)]
                .filter(Boolean)
                .join(" · ")}
            />
            <Divider />
            <SummaryRow label="Aktivitäten" value={activitySummary(draft)} />
            <Divider />
            <SummaryRow label="Rhythmus" value={rhythmSummary(draft)} />
            <Divider />
            <SummaryRow label="Feste Tage" value={structureSummary(draft)} />
          </Card>
          <Text variant="caption">Turniere und Reitbeteiligungen legst du später im Profil fest.</Text>
        </>
      ) : null}

      <View className="gap-3">
        {error ? (
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        {step < 4 ? (
          <Button
            label="Weiter"
            size="lg"
            fullWidth
            disabled={stepError !== null}
            onPress={() => goTo((step + 1) as Step)}
          />
        ) : (
          <Button label="Profil anlegen" size="lg" fullWidth loading={save.isPending} onPress={submit} />
        )}
        {step > 0 ? (
          <Button label="Zurück" variant="ghost" fullWidth onPress={() => goTo((step - 1) as Step)} />
        ) : null}
      </View>
    </Screen>
  );
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <View className="gap-0.5 px-4 py-3">
      <Text variant="label">{label}</Text>
      <Text variant="body">{value}</Text>
    </View>
  );
}
