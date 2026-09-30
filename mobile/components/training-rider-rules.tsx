import { View } from "react-native";

import { Card, Pill, Switch, Text } from "@/components/ui";
import { ACTIVITIES, MAX_INTENSITY_OPTIONS, activityLabel } from "@/lib/training";
import { toggleActivity, type RiderDraft } from "@/lib/training-profile";

export type RiderRuleCardProps = {
  rider: RiderDraft;
  /** False shows the rules without the ability to change them (the rider's own view). */
  editable: boolean;
  onChange?: (patch: Partial<RiderDraft>) => void;
};

/** What one rider may do on the horse: activities, highest load, hacking alone and shows. */
export function RiderRuleCard({ rider, editable, onChange }: RiderRuleCardProps) {
  const change = (patch: Partial<RiderDraft>) => onChange?.(patch);
  return (
    <Card className="gap-3">
      <Text variant="bodyStrong">{rider.name || "Reitbeteiligung"}</Text>
      <Text variant="label">Erlaubte Aktivitäten</Text>
      <View className="flex-row flex-wrap gap-2">
        {ACTIVITIES.map((a) => (
          <Pill
            key={a}
            label={activityLabel(a)}
            selected={rider.activities.includes(a)}
            disabled={!editable}
            onPress={() => change({ activities: toggleActivity(rider.activities, a) })}
          />
        ))}
      </View>
      <Text variant="label">Höchste Belastung</Text>
      <View className="flex-row flex-wrap gap-2">
        {MAX_INTENSITY_OPTIONS.map((o) => (
          <Pill
            key={o.value}
            label={o.label}
            selected={rider.maxIntensity === o.value}
            disabled={!editable}
            onPress={() => change({ maxIntensity: o.value })}
          />
        ))}
      </View>
      <Switch
        label="Darf allein ausreiten"
        value={rider.mayHackAlone}
        disabled={!editable}
        onValueChange={(mayHackAlone) => change({ mayHackAlone })}
      />
      <Switch
        label="Darf Turniere reiten"
        value={rider.mayRideShows}
        disabled={!editable}
        onValueChange={(mayRideShows) => change({ mayRideShows })}
      />
    </Card>
  );
}
