import { View } from "react-native";

import { cn } from "@/lib/cn";

import { Text } from "./text";

export type StepperProps = {
  /** Total number of steps. */
  steps: number;
  /** Zero-based index of the current step. */
  current: number;
  /** German title of the current step, shown after "Schritt 2 von 5". */
  label?: string;
  className?: string;
};

/**
 * Progress of a multi-step flow: one bar per step (done and current in `primary`) and a caption
 * like "Schritt 2 von 5 · Aktivitäten" below. Goes inside `PageHeader` of the wizard screens.
 *
 * @example <Stepper steps={5} current={1} label="Aktivitäten" />
 */
export function Stepper({ steps, current, label, className }: StepperProps) {
  const caption = `Schritt ${current + 1} von ${steps}`;
  return (
    <View
      accessible
      accessibilityRole="progressbar"
      accessibilityLabel={label ? `${caption}, ${label}` : caption}
      accessibilityValue={{ min: 1, max: steps, now: current + 1 }}
      className={cn("gap-2", className)}
    >
      <View className="flex-row gap-1.5">
        {Array.from({ length: steps }, (_, i) => (
          <View key={i} className={cn("h-1 flex-1 rounded-pill", i <= current ? "bg-primary" : "bg-border")} />
        ))}
      </View>
      <Text variant="caption">{label ? `${caption} · ${label}` : caption}</Text>
    </View>
  );
}
