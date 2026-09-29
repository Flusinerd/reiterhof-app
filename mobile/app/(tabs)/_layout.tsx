import { Tabs } from "expo-router";
import { Activity, HandHelping, House, PawPrint, Shirt } from "lucide-react-native";
import type { LucideIcon } from "lucide-react-native";
import { View, type ColorValue } from "react-native";

import { Icon } from "@/components/ui/icon";
import { cn } from "@/lib/cn";
import { colors, tabBar } from "@/lib/theme";

/** Active tab: green icon inside a primary-soft pill. */
function tabIcon(icon: LucideIcon) {
  return function TabIcon({ focused, color }: { focused: boolean; color: ColorValue }) {
    return (
      <View
        className={cn("items-center justify-center rounded-pill", focused && "bg-primary-soft")}
        style={{ width: tabBar.pill.width, height: tabBar.pill.height }}
      >
        <Icon as={icon} size={tabBar.iconSize} color={String(color)} />
      </View>
    );
  };
}

export default function TabsLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: tabBar.activeTint,
        tabBarInactiveTintColor: tabBar.inactiveTint,
        tabBarLabelStyle: { fontFamily: tabBar.labelFont, fontSize: tabBar.labelSize },
        tabBarStyle: {
          height: tabBar.height,
          paddingTop: tabBar.paddingTop,
          paddingBottom: tabBar.paddingBottom,
          backgroundColor: tabBar.background,
          borderTopColor: tabBar.borderColor,
          borderTopWidth: 1,
          elevation: 0,
          shadowOpacity: 0,
        },
        sceneStyle: { backgroundColor: colors.background },
      }}
    >
      <Tabs.Screen name="index" options={{ title: "Start", tabBarIcon: tabIcon(House) }} />
      <Tabs.Screen name="blankets" options={{ title: "Decken", tabBarIcon: tabIcon(Shirt) }} />
      <Tabs.Screen
        name="requests"
        options={{ title: "Anfragen", tabBarIcon: tabIcon(HandHelping) }}
      />
      <Tabs.Screen name="training" options={{ title: "Training", tabBarIcon: tabIcon(Activity) }} />
      <Tabs.Screen name="horses" options={{ title: "Pferde", tabBarIcon: tabIcon(PawPrint) }} />
    </Tabs>
  );
}
