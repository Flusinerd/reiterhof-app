import { View, type ViewProps } from "react-native";

import { colorsForKey } from "@/lib/color-keys";
import { initialOf } from "@/lib/initials";

import { Text } from "./text";

const SIZES = {
  sm: { box: 32, font: 14 },
  md: { box: 44, font: 18 },
  lg: { box: 64, font: 26 },
  xl: { box: 96, font: 40 },
} as const;

export type AvatarSize = keyof typeof SIZES;

export type AvatarProps = Omit<ViewProps, "children"> & {
  /** Name of the horse or person; its first letter is shown. */
  name: string;
  /** `color_key` from the backend (green, amber, blue, rose, violet, teal, neutral). Unknown/missing falls back to neutral. */
  colorKey?: string | null;
  size?: AvatarSize;
  className?: string;
};

/**
 * Round avatar with the initial of the name on the palette color of `colorKey`.
 *
 * @example <Avatar name="Luna" colorKey="green" size="lg" />
 */
export function Avatar({ name, colorKey, size = "md", className, style, ...props }: AvatarProps) {
  const { bg, fg } = colorsForKey(colorKey);
  const { box, font } = SIZES[size];
  return (
    <View
      accessibilityRole="image"
      accessibilityLabel={name}
      className={className}
      style={[
        {
          width: box,
          height: box,
          borderRadius: box / 2,
          backgroundColor: bg,
          alignItems: "center",
          justifyContent: "center",
        },
        style,
      ]}
      {...props}
    >
      <Text
        variant="bodyStrong"
        className="font-display"
        style={{ color: fg, fontSize: font, lineHeight: font * 1.2 }}
      >
        {initialOf(name)}
      </Text>
    </View>
  );
}
