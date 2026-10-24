import { router, usePathname, type Href } from "expo-router";
import { Bell, LogIn, LogOut, Settings } from "lucide-react-native";
import { useEffect } from "react";
import { Alert, View } from "react-native";

import { usePresenceCheckIn } from "@/components/presence-check-in";
import { Button } from "@/components/ui";

/** "Bin da" / "Ich gehe": checks in or out at the stable. A long press opens the presence screen. */
export function PresenceButton() {
  const presence = usePresenceCheckIn();

  useEffect(() => {
    if (presence.error) Alert.alert("Das hat nicht geklappt", presence.error);
  }, [presence.error]);

  return (
    <>
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
      />
      {presence.sheet}
    </>
  );
}

/**
 * The menu top right of sub pages: check in or out at the stable, the reminders, the settings.
 * The button of the screen that is open is left out. Tab screens have no menu: "Stall" and "Ich"
 * carry these.
 */
export function AppMenu() {
  const pathname = usePathname();
  return (
    <View className="flex-row items-center gap-1">
      {pathname !== "/presence" ? (
        <View className="mr-1">
          <PresenceButton />
        </View>
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
    </View>
  );
}
