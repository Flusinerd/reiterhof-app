import { Tabs } from "expo-router";

export default function TabsLayout() {
  return (
    <Tabs>
      <Tabs.Screen name="index" options={{ title: "Start" }} />
      <Tabs.Screen name="blankets" options={{ title: "Decken" }} />
      <Tabs.Screen name="requests" options={{ title: "Anfragen" }} />
      <Tabs.Screen name="training" options={{ title: "Training" }} />
      <Tabs.Screen name="horses" options={{ title: "Pferde" }} />
    </Tabs>
  );
}
