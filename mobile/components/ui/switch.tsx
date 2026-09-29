import { Pressable, Switch as RNSwitch, View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";
import { switchColors } from "@/lib/theme";

import { Text } from "./text";

export type SwitchProps = Omit<ViewProps, "children"> & {
  value: boolean;
  onValueChange: (value: boolean) => void;
  /** German label shown left of the switch; the whole row is tappable (>= 44 px). */
  label?: string;
  /** Optional secondary line below the label. */
  description?: string;
  disabled?: boolean;
  className?: string;
};

/**
 * On/off toggle in the primary color.
 *
 * @example <Switch label="Erinnerungen" value={on} onValueChange={setOn} />
 */
export function Switch({
  value,
  onValueChange,
  label,
  description,
  disabled = false,
  className,
  accessibilityLabel,
  ...props
}: SwitchProps) {
  return (
    <View className={cn("min-h-touch flex-row items-center justify-between gap-4", className)} {...props}>
      {label ? (
        <Pressable
          className="flex-1"
          disabled={disabled}
          onPress={() => onValueChange(!value)}
          accessible={false}
        >
          <Text variant="body" className={disabled ? "opacity-50" : undefined}>
            {label}
          </Text>
          {description ? <Text variant="secondary">{description}</Text> : null}
        </Pressable>
      ) : null}
      <RNSwitch
        value={value}
        onValueChange={onValueChange}
        disabled={disabled}
        accessibilityLabel={accessibilityLabel ?? label}
        trackColor={{ false: switchColors.trackOff, true: switchColors.trackOn }}
        thumbColor={switchColors.thumb}
        ios_backgroundColor={switchColors.trackOff}
      />
    </View>
  );
}
