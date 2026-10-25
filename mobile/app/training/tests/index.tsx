import { useLocalSearchParams } from "expo-router";
import { ExternalLink, Plus } from "lucide-react-native";
import { ActivityIndicator, Linking, View } from "react-native";

import { Button, Icon, PageHeader, Screen, Section, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { dressageError, isForbidden, isUnavailable, useDressageTests } from "@/lib/api/dressage";
import { OFFICIAL_LINKS, arenaLabel, defaultTitle, formatSeconds, spokenSeconds } from "@/lib/dressage";
import { pushOnce } from "@/lib/navigation";
import { useColors } from "@/lib/theme";
import { usePullToRefresh } from "@/lib/use-pull-to-refresh";

/** "Aufgaben und Küren": the saved tests of the horse, "Neue Aufgabe", and the links to the official sources. */
export default function DressageTests() {
  const { horse } = useLocalSearchParams<{ horse?: string }>();
  const c = useColors();
  const tests = useDressageTests(horse);
  const refresh = usePullToRefresh([tests]);

  const openNew = () => pushOnce({ pathname: "/training/tests/new", params: { horse } });

  return (
    <Screen back refreshControl={refresh}>
      <PageHeader
        eyebrow="Training"
        title="Aufgaben und Küren"
        description="Erstelle eine Aufgabe zum Reiten zu Hause oder plane deine Kür, mit Linien im Viereck und Zeit."
      />

      <Button label="Neue Aufgabe" icon={Plus} fullWidth onPress={openNew} disabled={!horse || (tests.isError && (isUnavailable(tests.error) || isForbidden(tests.error)))} />

      {tests.isPending && horse ? <ActivityIndicator accessibilityLabel="Lädt" color={c.primary.DEFAULT} className="self-start" /> : null}

      {tests.isError ? (
        <View className="gap-3">
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {isUnavailable(tests.error) ? "Das gibt es auf deinem Server noch nicht." : dressageError(tests.error)}
          </Text>
          {!isUnavailable(tests.error) && !isForbidden(tests.error) ? <Button label="Erneut versuchen" variant="outline" onPress={() => void tests.refetch()} /> : null}
        </View>
      ) : null}

      {tests.isSuccess ? (
        tests.data.length === 0 ? (
          <Text variant="body" tone="muted">
            Noch nichts gespeichert.
          </Text>
        ) : (
          <Section title="Gespeichert">
            <RowList>
              {tests.data.map((t) => {
                const title = t.title || defaultTitle(t.kind, t.class, t.arena);
                const detail = `Klasse ${t.class} · ${arenaLabel(t.arena)} · ${formatSeconds(t.seconds)} Min.`;
                return (
                  <ListRow
                    key={t.id}
                    eyebrow={t.kind === "freestyle" ? "Kür" : "Übungsaufgabe"}
                    title={title}
                    detail={detail}
                    accessibilityLabel={`${title}, Klasse ${t.class}, ${arenaLabel(t.arena)}, ${spokenSeconds(t.seconds)}`}
                    onPress={() => pushOnce({ pathname: "/training/tests/[id]", params: { id: t.id, horse } })}
                  />
                );
              })}
            </RowList>
          </Section>
        )
      ) : null}

      <Section title="Offizielle Aufgaben" description="Die Aufgabenhefte und das Kür-Merkblatt gibt es bei der FN. Die App zeigt sie nicht.">
        <RowList>
          {OFFICIAL_LINKS.map((l) => (
            <ListRow
              key={l.url}
              leading={<Icon as={ExternalLink} size={20} className="text-muted" />}
              title={l.title}
              detail={l.detail}
              chevron={false}
              accessibilityLabel={`${l.title}, ${l.detail}, öffnet im Browser`}
              onPress={() => void Linking.openURL(l.url)}
            />
          ))}
        </RowList>
      </Section>
    </Screen>
  );
}
