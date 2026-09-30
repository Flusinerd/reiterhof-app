import { Tabs } from "expo-router";
import { Activity, HandHelping, House, Shirt } from "lucide-react-native";
import type { LucideIcon } from "lucide-react-native";
import { Text, View, type ColorValue } from "react-native";

import { Horse } from "@/components/icons/horse";
import { Icon } from "@/components/ui/icon";
import { cn } from "@/lib/cn";
import { colors, tabBar } from "@/lib/theme";

/** Tab item: icon and label together; the active tab gets a primary-soft pill behind both. */
function tabIcon(icon: LucideIcon, label: string) {
  return function TabIcon({ focused, color }: { focused: boolean; color: ColorValue }) {
    return (
      <View
        className={cn("items-center justify-center gap-0.5 rounded-tile", focused && "bg-primary-soft")}
        style={{ width: tabBar.pill.width, height: tabBar.pill.height }}
      >
        <Icon as={icon} size={tabBar.iconSize} color={String(color)} />
        <Text
          numberOfLines={1}
          style={{ color: String(color), fontFamily: tabBar.labelFont, fontSize: tabBar.labelSize, lineHeight: 16 }}
        >
          {label}
        </Text>
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
        tabBarShowLabel: false,
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
      <Tabs.Screen name="index" options={{ title: "Start", tabBarIcon: tabIcon(House, "Start") }} />
      <Tabs.Screen name="blankets" options={{ title: "Decken", tabBarIcon: tabIcon(Shirt, "Decken") }} />
      <Tabs.Screen
        name="requests"
        options={{ title: "Anfragen", tabBarIcon: tabIcon(HandHelping, "Anfragen") }}
      />
      <Tabs.Screen name="training" options={{ title: "Training", tabBarIcon: tabIcon(Activity, "Training") }} />
      <Tabs.Screen name="horses" options={{ title: "Pferde", tabBarIcon: tabIcon(Horse, "Pferde") }} />
    </Tabs>
  );
}
