import { Shirt } from "lucide-react-native";
import { Image, View } from "react-native";

import { Icon } from "@/components/ui";
import { fileSource } from "@/lib/upload";

/** Photo of a blanket, or a neutral tile with a blanket icon when there is none. */
export function BlanketPhoto({ url, size = 56 }: { url: string | null; size?: number }) {
  if (url) {
    return (
      <Image
        source={fileSource(url)}
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
