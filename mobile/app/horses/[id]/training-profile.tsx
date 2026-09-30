import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { Activity, CalendarRange, Medal, Timer, Trophy, Users } from "lucide-react-native";
import { useEffect, useMemo, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { RiderRuleCard } from "@/components/training-rider-rules";
import {
  Badge,
  Button,
  Card,
  Divider,
  Input,
  LinkRow,
  PageHeader,
  Pill,
  Screen,
  Section,
  Sheet,
  Text,
} from "@/components/ui";
import { trainingError, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import { dateToIso } from "@/lib/date-field";
import { horseRoutes } from "@/lib/horse-format";
import { ACTIVITIES, DISCIPLINES, PROFILE_STATUSES, activityLabel, disciplineLabel, statusLabel } from "@/lib/training";
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

type EditorSection = "activities" | "rhythm" | "structure" | "shows" | "riders";

const STATUS_BADGE = { fit: "primary", reha: "accent", pause: "neutral" } as const;

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
        <Card className="gap-3">
          <Text variant="secondary">{profile.isError ? trainingError(profile.error) : "Kein Profil gefunden."}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void profile.refetch()} />
        </Card>
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
          <Badge variant={STATUS_BADGE[draft.status]} label={statusLabel(draft.status)} />
        ) : (
          <Badge variant="accent" label="Noch nicht angelegt" />
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

          <Card padded={false}>
            <LinkRow icon={Medal} label="Disziplin und Niveau" description={disciplineLine(draft)} onPress={openSheet} />
            <Divider />
            <LinkRow icon={Activity} label="Aktivitäten" description={activitySummary(draft)} onPress={() => go("activities")} />
            <Divider />
            <LinkRow icon={Timer} label="Rhythmus" description={rhythmSummary(draft)} onPress={() => go("rhythm")} />
            <Divider />
            <LinkRow
              icon={CalendarRange}
              label="Feste Tage und Ziele"
              description={structureSummary(draft)}
              onPress={() => go("structure")}
            />
            <Divider />
            <LinkRow
              icon={Trophy}
              label="Turniere"
              description={showsSummary(draft.shows, dateToIso(new Date()))}
              onPress={() => go("shows")}
            />
            {draft.riders.length > 0 ? (
              <>
                <Divider />
                <LinkRow
                  icon={Users}
                  label="Reitbeteiligungen"
                  description={ridersSummary(draft.riders)}
                  onPress={() => go("riders")}
                />
              </>
            ) : null}
          </Card>

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
              <Text variant="label">Niveau</Text>
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
        </>
      ) : (
        <>
          <Text variant="body">
            {[disciplineLine(draft), allowedLine(draft)].filter(Boolean).join(" · ")}
          </Text>
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
