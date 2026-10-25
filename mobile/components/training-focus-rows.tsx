import { View } from "react-native";

import { Section, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { focusSections, type FocusEntry } from "@/lib/training-focus";

/** The position of a chosen focus (1, 2, 3) on the primary surface, or an empty ring. */
function Marker({ position }: { position: number | null }) {
  if (position === null) return <View className="h-7 w-7 rounded-pill border border-border" />;
  return (
    <View className="h-7 w-7 items-center justify-center rounded-pill bg-primary">
      <Text variant="caption" className="font-sans-semibold text-primary-foreground">
        {position}
      </Text>
    </View>
  );
}

/**
 * The focus catalog as rows, grouped "Skala der Ausbildung" and "Praxis": label, what it is and
 * how you notice it. A tap chooses or drops a focus; chosen ones show their position. Shared by
 * the profile editor (up to three focuses of the horse) and the sheet "Mein Schwerpunkt" (two).
 */
export function FocusRows({
  catalog,
  selected,
  onToggle,
  notes,
}: {
  catalog: readonly FocusEntry[];
  /** Chosen keys in priority order. */
  selected: readonly string[];
  onToggle: (key: string) => void;
  /** A quiet line under a row, by key (warning or hint). */
  notes?: Readonly<Record<string, string>>;
}) {
  return (
    <View className="gap-6">
      {focusSections(catalog).map((section) => (
        <Section key={section.group} title={section.label}>
          <RowList>
            {section.entries.map((e) => {
              const index = selected.indexOf(e.key);
              const note = notes?.[e.key];
              return (
                <ListRow
                  key={e.key}
                  leading={<Marker position={index >= 0 ? index + 1 : null} />}
                  title={e.label}
                  detail={e.description}
                  chevron={false}
                  accessibilityLabel={[e.label, index >= 0 ? `gewählt, Platz ${index + 1}` : null, e.description]
                    .filter(Boolean)
                    .join(", ")}
                  onPress={() => onToggle(e.key)}
                >
                  {e.signs ? (
                    <Text variant="caption">
                      <Text variant="caption" className="font-sans-semibold">
                        Woran du es merkst:
                      </Text>{" "}
                      {e.signs}
                    </Text>
                  ) : null}
                  {note ? (
                    <Text variant="secondary" tone="accent">
                      {note}
                    </Text>
                  ) : null}
                </ListRow>
              );
            })}
          </RowList>
        </Section>
      ))}
    </View>
  );
}
