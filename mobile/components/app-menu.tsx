import { router, usePathname, type Href } from "expo-router";
import { Bell, LogIn, LogOut, Settings } from "lucide-react-native";
import { useEffect } from "react";
import { Alert, View } from "react-native";

import { usePresenceCheckIn } from "@/components/presence-check-in";
import { Button } from "@/components/ui";

/**
 * The menu top right on every screen: check in or out at the stable, the reminders, the settings.
 * A long press on the presence button opens the presence screen. The button of the screen that is
 * open is left out.
 */
export function AppMenu() {
  const pathname = usePathname();
  const presence = usePresenceCheckIn();

  useEffect(() => {
    if (presence.error) Alert.alert("Das hat nicht geklappt", presence.error);
  }, [presence.error]);

  return (
    <View className="flex-row items-center gap-1">
      {pathname !== "/presence" ? (
        <Button
          label={presence.here ? "Ich gehe" : "Bin da"}
          icon={presence.here ? LogOut : LogIn}
          variant={presence.here ? "secondary" : "outline"}
          size="sm"
          accessibilityLabel={presence.here ? "Im Stall. Abmelden" : "Nicht im Stall. Anmelden"}
          accessibilityHint="Lange drücken zeigt, wer im Stall ist."
          loading={presence.toggling}
          disabled={presence.overview.isPending}
          onPress={() => void presence.arriveOrLeave(!presence.here)}
          onLongPress={() => router.push("/presence")}
          className="mr-1"
        />
      ) : null}
      {pathname !== "/reminders" ? (
        <Button
          size="icon"
          variant="ghost"
          icon={Bell}
          accessibilityLabel="Erinnerungen"
          onPress={() => router.push("/reminders" as Href)}
        />
      ) : null}
      {pathname !== "/settings" ? (
        <Button
          size="icon"
          variant="ghost"
          icon={Settings}
          accessibilityLabel="Einstellungen"
          onPress={() => router.push("/settings" as Href)}
        />
      ) : null}
      {presence.sheet}
    </View>
  );
}
