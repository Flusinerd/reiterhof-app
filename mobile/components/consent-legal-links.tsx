import { useRouter } from "expo-router";
import { View } from "react-native";

import { Button } from "@/components/ui";
import { cn } from "@/lib/cn";

/** "Datenschutz" and "Lizenzen" links for screens shown before a session exists. */
export function LegalLinks({ className }: { className?: string }) {
  const router = useRouter();
  return (
    <View className={cn("flex-row flex-wrap justify-center gap-2", className)}>
      <Button label="Datenschutz" variant="ghost" size="sm" onPress={() => router.push("/legal/privacy")} />
      <Button label="Lizenzen" variant="ghost" size="sm" onPress={() => router.push("/legal/licenses")} />
    </View>
  );
}
