import { View } from "react-native";

import { Button } from "@/components/ui";
import { actionLabel, stateActions, type Recommendation, type StateAction } from "@/lib/blankets";

/**
 * The buttons for a horse's day state: "Eingedeckt" / "Abgedeckt" when the plan recommends a
 * blanket, "Geprüft" when it does not. The first one is the primary action. Text only and 128 px
 * wide at least: two fit next to each other in a card on a phone, otherwise they wrap into rows
 * of their own instead of cutting the label ("Eingedeckt" used to be clipped with an icon).
 */
export function BlanketActions({
  recommendation,
  pending,
  disabled,
  onAction,
}: {
  recommendation: Recommendation;
  /** The action that is being sent right now (shows a spinner). */
  pending: StateAction | null;
  disabled?: boolean;
  onAction: (action: StateAction) => void;
}) {
  const actions = stateActions(recommendation);
  return (
    <View className="flex-row flex-wrap gap-3">
      {actions.map((action, i) => (
        <Button
          key={action}
          label={actionLabel(action)}
          variant={i === 0 ? "primary" : "outline"}
          loading={pending === action}
          disabled={disabled}
          onPress={() => onAction(action)}
          className="min-w-[128px] grow basis-[128px] px-3"
        />
      ))}
    </View>
  );
}
