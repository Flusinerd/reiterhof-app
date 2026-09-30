import { useRouter } from "expo-router";
import { View } from "react-native";

import { Button } from "@/components/ui";
import { cn } from "@/lib/cn";

/** "Datenschutz" and "Impressum" links for screens shown before a session exists. */
export function LegalLinks({ className }: { className?: string }) {
  const router = useRouter();
  return (
    <View className={cn("flex-row justify-center gap-2", className)}>
      <Button label="Datenschutz" variant="ghost" size="sm" onPress={() => router.push("/legal/privacy")} />
      <Button label="Impressum" variant="ghost" size="sm" onPress={() => router.push("/legal/imprint")} />
    </View>
  );
}
