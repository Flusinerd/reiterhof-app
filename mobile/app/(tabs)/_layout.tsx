import { Tabs } from "expo-router";
import { House, User, Warehouse } from "lucide-react-native";
import type { LucideIcon } from "lucide-react-native";
import { useEffect } from "react";
import { Text, View, type ColorValue } from "react-native";
import Animated, { useAnimatedStyle, useSharedValue, withTiming } from "react-native-reanimated";

import { Horse } from "@/components/icons/horse";
import { Icon } from "@/components/ui/icon";
import { useTrainingHorses } from "@/lib/api/training";
import { tabBar, useColors } from "@/lib/theme";

const MARK_MS = 200;

/** Tab item: icon and label; the active tab gets a short brass line above the icon. */
function TabItem({ icon, label, focused, color }: { icon: LucideIcon; label: string; focused: boolean; color: ColorValue }) {
  const progress = useSharedValue(focused ? 1 : 0);
  useEffect(() => {
    progress.value = withTiming(focused ? 1 : 0, { duration: MARK_MS });
  }, [focused, progress]);
  const mark = useAnimatedStyle(() => ({ opacity: progress.value, transform: [{ scaleX: 0.4 + 0.6 * progress.value }] }));
  return (
    <View className="w-20 items-center gap-1 pt-1">
      <Animated.View className="absolute -top-2 h-0.5 w-6 rounded-pill" style={[{ backgroundColor: String(color) }, mark]} />
      <Icon as={icon} size={tabBar.iconSize} color={String(color)} />
      <Text
        numberOfLines={1}
        style={{ color: String(color), fontFamily: tabBar.labelFont, fontSize: tabBar.labelSize, lineHeight: 14 }}
      >
        {label}
      </Text>
    </View>
  );
}

function tabIcon(icon: LucideIcon, label: string) {
  return function TabIcon({ focused, color }: { focused: boolean; color: ColorValue }) {
    return <TabItem icon={icon} label={label} focused={focused} color={color} />;
  };
}

/**
 * Heute, the user's horse (named after it when there is exactly one), Stall, Ich. Blankets,
 * requests, training and the horse list are sub pages (same URLs as before, so pushes still work).
 */
export default function TabsLayout() {
  const c = useColors();
  const mine = useTrainingHorses().data ?? [];
  const horseLabel = mine.length === 1 ? mine[0].name : "Pferde";
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        animation: "shift",
        transitionSpec: { animation: "timing", config: { duration: 220 } },
        tabBarActiveTintColor: c.brass.DEFAULT,
        tabBarInactiveTintColor: c.muted,
        tabBarShowLabel: false,
        tabBarStyle: {
          height: tabBar.height,
          paddingTop: tabBar.paddingTop,
          paddingBottom: tabBar.paddingBottom,
          backgroundColor: c.background,
          borderTopColor: c.divider,
          borderTopWidth: 1,
          elevation: 0,
          shadowOpacity: 0,
        },
        sceneStyle: { backgroundColor: c.background },
      }}
    >
      <Tabs.Screen name="index" options={{ title: "Heute", tabBarIcon: tabIcon(House, "Heute") }} />
      <Tabs.Screen name="horse" options={{ title: horseLabel, tabBarIcon: tabIcon(Horse, horseLabel) }} />
      <Tabs.Screen name="stable" options={{ title: "Stall", tabBarIcon: tabIcon(Warehouse, "Stall") }} />
      <Tabs.Screen name="me" options={{ title: "Ich", tabBarIcon: tabIcon(User, "Ich") }} />
    </Tabs>
  );
}
