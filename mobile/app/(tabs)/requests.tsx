import { StyleSheet, Text, View } from "react-native";

export default function Requests() {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>Anfragen</Text>
      <Text>Anfragen an die Stallgasse folgen in Meilenstein 4.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8 },
  title: { fontSize: 22, fontWeight: "600" },
});
