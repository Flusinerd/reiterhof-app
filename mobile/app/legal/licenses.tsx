import { useMemo } from "react";
import { FlatList, View } from "react-native";

import { PageHeader, Screen, Text } from "@/components/ui";
import { LICENSE_TEXTS, LICENSES } from "@/lib/licenses.generated";

type Notice = { text: string; packages: string[] };

/**
 * Open-source notices (Rechtliches > Lizenzen). Generated at install time by
 * scripts/gen-licenses.mjs from the installed packages: one block per distinct license text
 * with the packages it covers. Reachable without a session, like the other legal texts.
 */
export default function Licenses() {
  const notices = useMemo<Notice[]>(() => {
    const byText = new Map<number, string[]>();
    for (const l of LICENSES) {
      const list = byText.get(l.text) ?? [];
      list.push(`${l.name} ${l.version} (${l.license})`);
      byText.set(l.text, list);
    }
    return [...byText.entries()]
      .map(([index, packages]) => ({ text: LICENSE_TEXTS[index] ?? "", packages }))
      .sort((a, b) => (a.packages[0] ?? "").localeCompare(b.packages[0] ?? ""));
  }, []);

  return (
    <Screen back scroll={false}>
      <FlatList
        data={notices}
        keyExtractor={(_, i) => String(i)}
        initialNumToRender={6}
        windowSize={5}
        contentContainerClassName="gap-6 pb-8"
        ListHeaderComponent={
          <PageHeader
            eyebrow="Rechtliches"
            title="Open-Source-Lizenzen"
            description={`Stallfunk enthält Software anderer Autorinnen und Autoren (${LICENSES.length} Pakete). Hier stehen ihre Lizenzhinweise.`}
          />
        }
        renderItem={({ item }) => (
          <View className="gap-2 rounded-card border border-border bg-card p-4">
            <Text variant="bodyStrong">{item.packages.join(", ")}</Text>
            <Text variant="caption">{item.text}</Text>
          </View>
        )}
      />
    </Screen>
  );
}
