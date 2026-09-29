import { Tabs } from "expo-router";

export default function TabsLayout() {
  return (
    <Tabs>
      <Tabs.Screen name="index" options={{ title: "Stunden" }} />
      <Tabs.Screen name="pferde" options={{ title: "Pferde" }} />
      <Tabs.Screen name="mitteilungen" options={{ title: "Mitteilungen" }} />
    </Tabs>
  );
}
