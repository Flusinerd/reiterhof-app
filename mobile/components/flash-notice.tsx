import { Check } from "lucide-react-native";
import { useEffect, useSyncExternalStore } from "react";
import { View } from "react-native";

import { Button, Icon, Text } from "@/components/ui";
import { flash } from "@/lib/flash";

/** How long the result stays when nobody closes it. */
const SHOW_MS = 10_000;

/**
 * The result of the flow that led here ("Luna wurde gelöscht."): a check mark and one line in the
 * success tone, closed by "Schließen" or after ten seconds. Nothing when there is none.
 */
export function FlashNotice() {
  const text = useSyncExternalStore(flash.subscribe, flash.get, flash.get);
  useEffect(() => {
    if (!text) return;
    const timer = setTimeout(flash.clear, SHOW_MS);
    return () => clearTimeout(timer);
  }, [text]);
  if (!text) return null;
  return (
    <View className="flex-row items-center gap-2" accessibilityRole="alert" accessibilityLiveRegion="polite">
      <Icon as={Check} size={18} className="text-success" />
      <Text variant="bodyStrong" tone="success" className="flex-1">
        {text}
      </Text>
      <Button label="Schließen" variant="ghost" size="sm" onPress={flash.clear} />
    </View>
  );
}
