import { View } from "react-native";

import { ActivityIcon } from "@/components/training-activity-icon";
import { Badge, Button, Divider, Sheet, Text } from "@/components/ui";
import { formatDayLong, weekdayShort } from "@/lib/training";
import { canAskForAI, planStatusText, type PlanDay, type PlanResponse } from "@/lib/training-plan";

export type PlanSheetProps = {
  plan: PlanResponse | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onApply: (days: PlanDay[]) => void;
  applying: boolean;
  /** Opens the ai_training consent and plans again; only offered to the owner. */
  onAllowAI: () => void;
  error: string | null;
};

/**
 * Week plan proposal (JAN-89): one row per open day with activity, minutes and the reason.
 * Days from the language model carry the badge "KI-Vorschlag"; the rules checked them. Nothing
 * is stored until "Übernehmen".
 */
export function PlanSheet({ plan, open, onOpenChange, onApply, applying, onAllowAI, error }: PlanSheetProps) {
  const hint = plan ? planStatusText(plan.ai_status, plan.owner_is_me) : null;
  const days = plan?.days ?? [];
  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title="Vorschlag für die Woche"
      description={plan?.source === "ai" ? "Mit KI geplant, von den festen Regeln geprüft." : "Aus den festen Regeln."}
    >
      {hint ? <Text variant="secondary">{hint}</Text> : null}
      {plan && canAskForAI(plan) ? (
        <Button label="KI-Vorschläge erlauben" variant="outline" fullWidth disabled={applying} onPress={onAllowAI} />
      ) : null}

      {days.length > 0 ? (
        <View className="rounded-card border border-border">
          {days.map((d, i) => (
            <View key={d.date}>
              {i > 0 ? <Divider /> : null}
              <PlanRow day={d} />
            </View>
          ))}
        </View>
      ) : null}

      {plan?.source === "ai" ? (
        <Text variant="caption">
          KI-Vorschläge sind Vorschläge. Entscheide selbst, ob sie zu deinem Pferd passen; im Zweifel gilt, was Tierarzt oder
          Trainer sagen.
        </Text>
      ) : null}
      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      {days.length > 0 ? (
        <Button label="Übernehmen" fullWidth loading={applying} onPress={() => onApply(days)} />
      ) : null}
      <Button label={days.length > 0 ? "Verwerfen" : "Schließen"} variant="outline" fullWidth disabled={applying} onPress={() => onOpenChange(false)} />
    </Sheet>
  );
}

function PlanRow({ day }: { day: PlanDay }) {
  const what = day.activity === "rest" ? day.label : `${day.label} · ${day.minutes} Min.`;
  const who = !day.user ? "" : day.activity === "rest" ? ` · ${day.user.name} bleibt eingetragen` : ` · für ${day.user.name}`;
  return (
    <View className="gap-1.5 p-3" accessible accessibilityLabel={`${formatDayLong(day.date)}: ${what}${who}. ${day.reason}`}>
      <View className="flex-row items-center gap-3">
        <Text variant="bodyStrong" className="w-8">
          {weekdayShort(day.date)}
        </Text>
        <ActivityIcon activity={day.activity} size={18} />
        <Text variant="body" className="flex-1">
          {what}
          {who}
        </Text>
        <Badge variant={day.source === "ai" ? "info" : "neutral"} label={day.source === "ai" ? "KI-Vorschlag" : "Regel"} />
      </View>
      {day.reason ? (
        <Text variant="secondary" className="ml-11">
          {day.reason}
        </Text>
      ) : null}
      {day.note ? (
        <Text variant="caption" className="ml-11">
          Bedingung: {day.note}
        </Text>
      ) : null}
      {day.replaced ? (
        <Text variant="caption" className="ml-11">
          KI-Vorschlag ersetzt: {day.replaced}
        </Text>
      ) : null}
    </View>
  );
}
