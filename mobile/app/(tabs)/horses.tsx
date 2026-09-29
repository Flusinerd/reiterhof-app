import { StyleSheet, Text, View } from "react-native";

export default function Horses() {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>Pferde</Text>
      <Text>Pferdeakte und Notfallkarte folgen in Meilenstein 5.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8 },
  title: { fontSize: 22, fontWeight: "600" },
});
