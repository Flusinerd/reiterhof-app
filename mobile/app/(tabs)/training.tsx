import { StyleSheet, Text, View } from "react-native";

export default function Training() {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>Training</Text>
      <Text>Die Empfehlung „Was heute?“ folgt in Meilenstein 6.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8 },
  title: { fontSize: 22, fontWeight: "600" },
});
