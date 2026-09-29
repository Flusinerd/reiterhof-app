import { useQuery } from "@tanstack/react-query";
import { ActivityIndicator, StyleSheet, Text, View } from "react-native";

import { fetchHealth } from "../../lib/api";

export default function Stunden() {
  const health = useQuery({ queryKey: ["health"], queryFn: fetchHealth });

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Reitstunden</Text>
      <Text>Der Stundenplan folgt in Meilenstein 3.</Text>
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
