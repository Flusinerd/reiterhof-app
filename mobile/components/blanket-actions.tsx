import { View } from "react-native";

import { Button } from "@/components/ui";
import { actionLabel, stateActions, type Recommendation, type StateAction } from "@/lib/blankets";

/**
 * The buttons for a horse's day state: "Eingedeckt" / "Abgedeckt" when the plan recommends a
 * blanket, "Geprüft" when it does not, all three when there is no recommendation. They sit in one
 * row and share its width (`flex-1`, 44 px high, 8 px padding): at 390 px three labels fit without
 * clipping ("Eingedeckt" is the longest, about 80 px). The first one is the recommended action
 * (`secondary`), the others are outlined; the primary button of the screen is "Runde starten".
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
  /** May return a promise: the tapped button stays busy until it settles. */
  onAction: (action: StateAction) => void | Promise<unknown>;
}) {
  const actions = stateActions(recommendation);
  return (
    <View className="flex-row gap-2">
      {actions.map((action, i) => (
        <Button
          key={action}
          label={actionLabel(action)}
          variant={i === 0 ? "secondary" : "outline"}
          size="sm"
          loading={pending === action}
          disabled={disabled}
          onPress={() => onAction(action)}
          className="flex-1 px-2"
        />
      ))}
    </View>
  );
}
