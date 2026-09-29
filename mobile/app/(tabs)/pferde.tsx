import { StyleSheet, Text, View } from "react-native";

export default function Pferde() {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>Pferde</Text>
      <Text>Pferde und Boxen folgen in Meilenstein 2.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8 },
  title: { fontSize: 22, fontWeight: "600" },
});
