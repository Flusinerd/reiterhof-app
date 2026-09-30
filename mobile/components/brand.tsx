import { Image, View, type ViewProps } from "react-native";

import { Text } from "@/components/ui";
import { cn } from "@/lib/cn";

const logo = require("../assets/logo-256.png");

export type BrandProps = Omit<ViewProps, "children"> & {
  /** `md`: 44 px logo with the wordmark (sign-in). `lg`: 64 px logo, wordmark below. */
  size?: "md" | "lg";
  className?: string;
};

/**
 * Logo and wordmark, as in the login mail: the horseshoe on the left, "Stallfunk" in
 * Fraunces. Only for screens without a session (sign-in, magic link), not as a header.
 *
 * @example <Brand />
 */
export function Brand({ size = "md", className, ...props }: BrandProps) {
  const box = size === "lg" ? 64 : 44;
  return (
    <View
      accessibilityRole="header"
      accessibilityLabel="Stallfunk"
      className={cn(size === "lg" ? "items-start gap-3" : "flex-row items-center gap-3", className)}
      {...props}
    >
      <Image source={logo} style={{ width: box, height: box, borderRadius: box / 4 }} accessibilityIgnoresInvertColors />
      <Text variant={size === "lg" ? "title" : "heading"} className="text-primary-deeper">
        Stallfunk
      </Text>
    </View>
  );
}
