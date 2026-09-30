import { useRouter } from "expo-router";
import { View } from "react-native";

import { Button } from "@/components/ui";

/** "Datenschutz" and "Impressum" links for screens shown before a session exists. */
export function LegalLinks() {
  const router = useRouter();
  return (
    <View className="flex-row justify-center gap-2">
      <Button label="Datenschutz" variant="ghost" size="sm" onPress={() => router.push("/legal/privacy")} />
      <Button label="Impressum" variant="ghost" size="sm" onPress={() => router.push("/legal/imprint")} />
    </View>
  );
}
