import { Check, Shirt, Sun } from "lucide-react-native";
import type { LucideIcon } from "lucide-react-native";
import { View } from "react-native";

import { Button } from "@/components/ui";
import { actionLabel, stateActions, type Recommendation, type StateAction } from "@/lib/blankets";

const ICONS: Record<StateAction, LucideIcon> = { covered: Shirt, uncovered: Sun, checked: Check };

/**
 * The buttons for a horse's day state: "Eingedeckt" / "Abgedeckt" when the plan recommends a
 * blanket, "Geprüft" when it does not. The first one is the primary action.
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
          icon={ICONS[action]}
          variant={i === 0 ? "primary" : "outline"}
          loading={pending === action}
          disabled={disabled}
          onPress={() => onAction(action)}
          className="flex-1"
        />
      ))}
    </View>
  );
}
