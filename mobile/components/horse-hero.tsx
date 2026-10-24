import { Camera } from "lucide-react-native";
import { useState } from "react";
import { Pressable, useColorScheme, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { AuthImage } from "@/components/auth-image";
import { Icon, Text } from "@/components/ui";
import type { Horse } from "@/lib/api/horses";
import { colorsForKey } from "@/lib/color-keys";
import { heroEyebrow, heroSubline, photoActionLabel } from "@/lib/horse-page";
import { initialOf } from "@/lib/initials";
import { useColors } from "@/lib/theme";

const PHOTO_HEIGHT = 270;
const FIELD_HEIGHT = 240;

function rgba(hex: string, alpha: number): string {
  const n = parseInt(hex.slice(1), 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
}

/**
 * Fades the picture into the page background: stacked bands of the background color with rising
 * opacity (no gradient dependency, same on every platform). `height` is the share of the hero.
 */
function Fade({ height }: { height: number }) {
  const { background } = useColors();
  const bands = 48;
  return (
    <View pointerEvents="none" style={{ position: "absolute", left: 0, right: 0, bottom: 0, height: `${height * 100}%` }}>
      {Array.from({ length: bands }, (_, i) => (
        <View key={i} style={{ flex: 1, backgroundColor: rgba(background, Math.min(1, ((i + 1) / bands) ** 1.25)) }} />
      ))}
    </View>
  );
}

export type HorseHeroProps = {
  horse: Horse;
  /** Shows the photo button (owner or admin). */
  canEditPhoto: boolean;
  onEditPhoto: () => void;
  /** The hero starts at the very top of the screen, behind the status bar (the horse tab). */
  behindStatusBar: boolean;
};

/**
 * Head of the horse page, full-bleed: the photo fading into the background, or without a photo the
 * horse's color as a field with its big italic initial and a quiet "+ Foto hinzufügen". Eyebrow
 * ("Stute · 11 Jahre · Box 12"), name in display italic, owner line.
 */
export function HorseHero({ horse, canEditPhoto, onEditPhoto, behindStatusBar }: HorseHeroProps) {
  const insets = useSafeAreaInsets();
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  const top = behindStatusBar ? insets.top : 0;
  const [failedUrl, setFailedUrl] = useState<string | null>(null);
  const photo = horse.photo_url && failedUrl !== horse.photo_url ? horse.photo_url : null;
  const { bg, fg } = colorsForKey(horse.color_key, scheme);
  const eyebrow = heroEyebrow(horse, new Date());
  const subline = heroSubline(horse);
  const height = (photo ? PHOTO_HEIGHT : FIELD_HEIGHT) + top;

  const text = (
    <View className="gap-1 px-6 pb-4">
      {eyebrow ? <Text variant="eyebrow">{eyebrow}</Text> : null}
      <Text variant="display" accessibilityRole="header" className="text-[44px] leading-[48px]">
        {horse.name}
      </Text>
      {subline ? (
        <Text variant="bodySm" tone="muted">
          {subline}
        </Text>
      ) : null}
    </View>
  );

  return (
    <View className={behindStatusBar ? "-mx-6 -mt-4" : "-mx-6"} style={{ height }}>
      {photo ? (
        <View style={{ flex: 1, justifyContent: "flex-end", overflow: "hidden" }} className="bg-divider">
          <AuthImage
            url={photo}
            accessibilityLabel={`Foto von ${horse.name}`}
            accessibilityIgnoresInvertColors
            onError={() => setFailedUrl(photo)}
            resizeMode="cover"
            style={{ position: "absolute", top: 0, left: 0, right: 0, bottom: 0, width: "100%", height: "100%" }}
          />
          <Fade height={0.78} />
          {text}
          {canEditPhoto ? (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={photoActionLabel(true)}
              onPress={onEditPhoto}
              hitSlop={6}
              className="h-11 w-11 items-center justify-center rounded-pill active:opacity-70"
              style={{ position: "absolute", right: 16, top: top + 8, backgroundColor: "rgba(8, 10, 9, 0.5)" }}
            >
              <Icon as={Camera} size={20} className="text-white" />
            </Pressable>
          ) : null}
        </View>
      ) : (
        <View style={{ flex: 1, justifyContent: "flex-end", overflow: "hidden", backgroundColor: bg }}>
          <Text
            accessibilityElementsHidden
            importantForAccessibility="no"
            numberOfLines={1}
            className="font-display-italic"
            style={{ position: "absolute", right: -10, top: top + 4, fontSize: 200, lineHeight: 220, color: fg, opacity: 0.2 }}
          >
            {initialOf(horse.name)}
          </Text>
          <Fade height={0.45} />
          {text}
          {canEditPhoto ? (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="Foto hinzufügen"
              onPress={onEditPhoto}
              hitSlop={8}
              className="mx-6 mb-4 min-h-8 self-start rounded-pill border border-dashed border-brass/60 px-3 py-1.5 active:opacity-70"
              style={{ marginTop: -8 }}
            >
              <Text className="font-sans-semibold text-secondary text-brass">{photoActionLabel(false)}</Text>
            </Pressable>
          ) : null}
        </View>
      )}
    </View>
  );
}
