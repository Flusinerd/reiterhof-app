import { Check, Undo2 } from "lucide-react-native";
import { View } from "react-native";

import { Badge, Button, Hero, Text } from "@/components/ui";
import type { RehaView } from "@/lib/api/reha";
import { formatDate } from "@/lib/training";

type Props = {
  view: RehaView;
  /** True while "Heute erledigt" or its undo is being sent. */
  busy?: boolean;
  onDone?: () => void;
  onUndo?: () => void;
};

/**
 * The one hero of the reha screen: "Heute erlaubt" with activity, minutes and conditions, and the
 * "Heute erledigt" button for those who may mark the day. Everybody in the stable sees the rule
 * (members may be asked to exercise the horse); the rest of the plan is for owner and riders.
 */
export function RehaTodayHero({ view, busy, onDone, onUndo }: Props) {
  const { today, plan } = view;

  if (!view.has_active_plan) {
    return (
      <Hero
        tone="soft"
        eyebrow={view.horse_name}
        title="Kein Reha-Plan"
        description={"Es gelten die normalen Trainingsregeln."}
      />
    );
  }

  if (!today) {
    const upcoming = plan?.state === "upcoming";
    const finished = plan?.state === "finished";
    return (
      <Hero
        tone="soft"
        eyebrow={`${view.horse_name} · Heute erlaubt`}
        title="Heute keine Einheit"
        description={
          upcoming && plan
            ? `Der Plan startet am ${formatDate(plan.start_date)}. Bis dahin nur leichte Bewegung nach Absprache.`
            : finished
              ? "Alle Phasen abgeschlossen. Der Besitzer kann den Plan beenden oder anpassen."
              : "Heute nichts vorgesehen. Frag den Besitzer."
        }
      />
    );
  }

  const ramp =
    today.min_minutes !== today.max_minutes
      ? `Tag ${today.day_in_phase} von ${today.days_in_phase}: ${today.min_minutes} bis ${today.max_minutes} Minuten.`
      : `Tag ${today.day_in_phase} von ${today.days_in_phase}.`;
  const eyebrow = `${view.horse_name} · Heute erlaubt · Phase ${today.phase_index} von ${today.phases}`;

  if (today.rest) {
    return (
      <Hero
        eyebrow={eyebrow}
        title={today.phase}
        description={
          today.conditions
            ? `Heute keine Bewegung. ${today.conditions}`
            : `Heute keine Bewegung. Tag ${today.day_in_phase} von ${today.days_in_phase}.`
        }
      />
    );
  }

  return (
    <Hero
      eyebrow={eyebrow}
      title={`${today.activity_label}: ${today.phase}`}
      value={String(today.minutes)}
      valueSize="lg"
      unit="Minuten"
      description={today.conditions ? `Bedingung: ${today.conditions}` : ramp}
    >
      {today.conditions ? <Text variant="secondary" className="text-white/70">{ramp}</Text> : null}
      {view.can_mark_done ? (
        today.done ? (
          <View className="gap-3">
            <Badge variant="primary" label={today.done_by ? `Erledigt von ${today.done_by}` : "Heute erledigt"} className="self-start" />
            <Button label="Rückgängig" variant="secondary" size="sm" icon={Undo2} loading={busy} onPress={onUndo} />
          </View>
        ) : (
          <Button label="Heute erledigt" variant="secondary" size="lg" icon={Check} fullWidth loading={busy} onPress={onDone} />
        )
      ) : today.done ? (
        <Badge variant="primary" label="Heute erledigt" className="self-start" />
      ) : null}
    </Hero>
  );
}
