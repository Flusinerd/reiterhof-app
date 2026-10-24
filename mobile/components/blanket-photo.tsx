import { Shirt } from "lucide-react-native";
import { View } from "react-native";

import { AuthImage } from "@/components/auth-image";
import { Icon } from "@/components/ui";

/** Photo of a blanket, or a neutral tile with a blanket icon when there is none. */
export function BlanketPhoto({ url, size = 56 }: { url: string | null; size?: number }) {
  if (url) {
    return (
      <AuthImage
        url={url}
        accessibilityIgnoresInvertColors
        accessibilityLabel="Foto der Decke"
        style={{ width: size, height: size }}
        className="rounded-tile bg-divider"
        resizeMode="cover"
      />
    );
  }
  return (
    <View
      style={{ width: size, height: size }}
      className="items-center justify-center rounded-tile bg-primary-soft"
      accessibilityElementsHidden
    >
      <Icon as={Shirt} size={Math.round(size / 2.4)} className="text-primary" />
    </View>
  );
}
