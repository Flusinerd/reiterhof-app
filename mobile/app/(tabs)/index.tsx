import { useQuery } from "@tanstack/react-query";
import { ActivityIndicator, StyleSheet, Text, View } from "react-native";

import { fetchHealth } from "../../lib/api";

export default function Home() {
  const health = useQuery({ queryKey: ["health"], queryFn: fetchHealth });

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Start</Text>
      <Text>Wetter und Deckenempfehlung folgen in Meilenstein 3.</Text>
      {health.isPending ? (
        <ActivityIndicator />
      ) : (
        <Text>
          API: {health.isError ? "nicht erreichbar" : health.data.status}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8 },
  title: { fontSize: 22, fontWeight: "600" },
});
