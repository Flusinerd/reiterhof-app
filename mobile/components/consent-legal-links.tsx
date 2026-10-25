import { View } from "react-native";

import { Button } from "@/components/ui";
import { cn } from "@/lib/cn";
import { pushOnce } from "@/lib/navigation";

/** "Datenschutz" and "Lizenzen" links for screens shown before a session exists. */
export function LegalLinks({ className }: { className?: string }) {
  return (
    <View className={cn("flex-row flex-wrap justify-center gap-2", className)}>
      <Button label="Datenschutz" variant="ghost" size="sm" onPress={() => pushOnce("/legal/privacy")} />
      <Button label="Lizenzen" variant="ghost" size="sm" onPress={() => pushOnce("/legal/licenses")} />
    </View>
  );
}
