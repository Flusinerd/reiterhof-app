import { StyleSheet, Text, View } from "react-native";

export default function Blankets() {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>Decken</Text>
      <Text>Deckenplan und Tagesstatus folgen in Meilenstein 3.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8 },
  title: { fontSize: 22, fontWeight: "600" },
});
