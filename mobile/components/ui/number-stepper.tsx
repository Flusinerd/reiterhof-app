import { Minus, Plus } from "lucide-react-native";
import { View } from "react-native";

import { cn } from "@/lib/cn";

import { Button } from "./button";
import { Text } from "./text";

export type NumberStepperProps = {
  value: number;
  min: number;
  max: number;
  onChange: (value: number) => void;
  /** German accessibility label, e.g. "Einheiten pro Woche". */
  accessibilityLabel: string;
  /** Text for the value, e.g. `(v) => `${v} Min.``. Defaults to the plain number. */
  format?: (value: number) => string;
  className?: string;
};

/**
 * Whole-number input with "minus" and "plus" buttons and the value between them. The buttons
 * are disabled at `min` and `max`. Screen readers get one adjustable element (swipe up/down).
 *
 * @example <NumberStepper value={n} min={1} max={7} onChange={setN} accessibilityLabel="Einheiten pro Woche" />
 */
export function NumberStepper({
  value,
  min,
  max,
  onChange,
  accessibilityLabel,
  format,
  className,
}: NumberStepperProps) {
  const text = format ? format(value) : String(value);
  const canDecrease = value > min;
  const canIncrease = value < max;
  return (
    <View
      accessible
      accessibilityRole="adjustable"
      accessibilityLabel={accessibilityLabel}
      accessibilityValue={{ text }}
      accessibilityActions={[{ name: "increment" }, { name: "decrement" }]}
      onAccessibilityAction={(e) => {
        if (e.nativeEvent.actionName === "increment" && canIncrease) onChange(value + 1);
        if (e.nativeEvent.actionName === "decrement" && canDecrease) onChange(value - 1);
      }}
      className={cn("flex-row items-center gap-2 self-start", className)}
    >
      <Button
        size="icon"
        variant="outline"
        icon={Minus}
        accessibilityLabel="Weniger"
        disabled={!canDecrease}
        onPress={() => onChange(Math.max(min, value - 1))}
      />
      <Text variant="titleLg" className="min-w-10 px-1 text-center">
        {text}
      </Text>
      <Button
        size="icon"
        variant="outline"
        icon={Plus}
        accessibilityLabel="Mehr"
        disabled={!canIncrease}
        onPress={() => onChange(Math.min(max, value + 1))}
      />
    </View>
  );
}

export type RangeStepperProps = {
  /** German field label, e.g. "Einheiten pro Woche". */
  label: string;
  min: number;
  max: number;
  /** Smallest value either end can take. */
  lowerBound: number;
  /** Largest value either end can take. */
  upperBound: number;
  onChange: (min: number, max: number) => void;
  className?: string;
};

/**
 * A from/to range of two `NumberStepper`s joined by "bis". Raising the lower end above the upper
 * end drags the upper end along, and lowering the upper end below the lower end drags the lower one.
 *
 * @example <RangeStepper label="Einheiten pro Woche" min={4} max={5} lowerBound={1} upperBound={7} onChange={set} />
 */
export function RangeStepper({ label, min, max, lowerBound, upperBound, onChange, className }: RangeStepperProps) {
  return (
    <View className={cn("gap-2", className)}>
      <Text variant="label">{label}</Text>
      <View className="flex-row flex-wrap items-center gap-x-3 gap-y-2">
        <NumberStepper
          value={min}
          min={lowerBound}
          max={upperBound}
          onChange={(v) => onChange(v, Math.max(v, max))}
          accessibilityLabel={`${label}, mindestens`}
        />
        <Text variant="secondary">bis</Text>
        <NumberStepper
          value={max}
          min={lowerBound}
          max={upperBound}
          onChange={(v) => onChange(Math.min(v, min), v)}
          accessibilityLabel={`${label}, höchstens`}
        />
      </View>
    </View>
  );
}
