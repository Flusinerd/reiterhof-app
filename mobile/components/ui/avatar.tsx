import { useState } from "react";
import { useColorScheme, View, type ViewProps } from "react-native";

import { AuthImage } from "@/components/auth-image";
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
  /** Relative API path of a photo (`Horse.photo_url`); shown instead of the initial. */
  photoUrl?: string | null;
  size?: AvatarSize;
  className?: string;
};

/**
 * Round avatar with the photo, or the initial of the name on the palette color of `colorKey`
 * (a calmer, darker tint in the dark scheme).
 *
 * @example <Avatar name="Luna" colorKey="green" photoUrl={horse.photo_url} size="lg" />
 */
export function Avatar({ name, colorKey, photoUrl, size = "md", className, style, ...props }: AvatarProps) {
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  const { bg, fg } = colorsForKey(colorKey, scheme);
  const { box, font } = SIZES[size];
  const [failedUrl, setFailedUrl] = useState<string | null>(null);
  const showPhoto = !!photoUrl && failedUrl !== photoUrl;
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
          overflow: "hidden",
        },
        style,
      ]}
      {...props}
    >
      {showPhoto ? (
        <AuthImage
          url={photoUrl}
          accessibilityIgnoresInvertColors
          onError={() => setFailedUrl(photoUrl)}
          style={{ width: box, height: box }}
          resizeMode="cover"
        />
      ) : (
        <Text
          variant="bodyStrong"
          className="font-display"
          style={{ color: fg, fontSize: font, lineHeight: font * 1.2 }}
        >
          {initialOf(name)}
        </Text>
      )}
    </View>
  );
}
