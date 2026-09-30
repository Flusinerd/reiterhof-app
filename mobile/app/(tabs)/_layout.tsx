import { Tabs } from "expo-router";
import { Activity, HandHelping, House, Shirt } from "lucide-react-native";
import type { LucideIcon } from "lucide-react-native";
import { useEffect } from "react";
import { Text, View, type ColorValue } from "react-native";
import Animated, { useAnimatedStyle, useSharedValue, withTiming } from "react-native-reanimated";

import { Horse } from "@/components/icons/horse";
import { Icon } from "@/components/ui/icon";
import { colors, tabBar } from "@/lib/theme";

const PILL_MS = 200;

/** Tab item: icon and label together; the active tab gets a primary-soft pill that fades and grows in. */
function TabItem({ icon, label, focused, color }: { icon: LucideIcon; label: string; focused: boolean; color: ColorValue }) {
  const progress = useSharedValue(focused ? 1 : 0);
  useEffect(() => {
    progress.value = withTiming(focused ? 1 : 0, { duration: PILL_MS });
  }, [focused, progress]);
  const pill = useAnimatedStyle(() => ({
    opacity: progress.value,
    transform: [{ scale: 0.85 + 0.15 * progress.value }],
  }));
  return (
    <View className="items-center justify-center" style={{ width: tabBar.pill.width, height: tabBar.pill.height }}>
      <Animated.View className="absolute inset-0 rounded-tile bg-primary-soft" style={pill} />
      <View className="items-center gap-0.5">
        <Icon as={icon} size={tabBar.iconSize} color={String(color)} />
        <Text
          numberOfLines={1}
          style={{ color: String(color), fontFamily: tabBar.labelFont, fontSize: tabBar.labelSize, lineHeight: 16 }}
        >
          {label}
        </Text>
      </View>
    </View>
  );
}

function tabIcon(icon: LucideIcon, label: string) {
  return function TabIcon({ focused, color }: { focused: boolean; color: ColorValue }) {
    return <TabItem icon={icon} label={label} focused={focused} color={color} />;
  };
}

export default function TabsLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        animation: "shift",
        transitionSpec: { animation: "timing", config: { duration: 220 } },
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
