import { ChevronRight, UserPlus } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import {
  Avatar,
  Button,
  Card,
  Divider,
  Icon,
  PressableCard,
  Sheet,
  Switch,
  Text,
} from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { horsesApi, useHorseMutation, useMembers, type Horse, type Rider } from "@/lib/api/horses";
import { DEFAULT_RIDER_RULES, RIDER_RULES, rulesSummary, toggleRule } from "@/lib/horse-format";

type Editing = { userId: string; name: string; colorKey: string | null; rules: string[]; isNew: boolean };

/**
 * Riders (RB) of a horse with what each may do (positive list). Owner and admins can add
 * riders, switch rules on and off and remove riders; everybody else sees the list.
 */
export function HorseRiders({ horse }: { horse: Horse }) {
  const members = useMembers();
  const save = useHorseMutation((v: { userId: string; rules: string[] }) =>
    horsesApi.setRider(horse.id, v.userId, v.rules),
  );
  const remove = useHorseMutation((userId: string) => horsesApi.removeRider(horse.id, userId));
  const [picking, setPicking] = useState(false);
  const [editing, setEditing] = useState<Editing | null>(null);
  const [error, setError] = useState<string | null>(null);

  const candidates = (members.data ?? []).filter(
    (m) => m.id !== horse.owner?.id && !horse.riders.some((r) => r.user_id === m.id),
  );

  function edit(r: Rider) {
    setError(null);
    setEditing({ userId: r.user_id, name: r.name, colorKey: r.color_key, rules: r.rules, isNew: false });
  }

  async function run(action: () => Promise<unknown>) {
    setError(null);
    try {
      await action();
      setEditing(null);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <View className="gap-3">
      {horse.riders.length === 0 ? (
        <Card>
          <Text variant="secondary">Keine Reitbeteiligungen eingetragen.</Text>
        </Card>
      ) : (
        horse.riders.map((r) => {
          const row = (
            <>
              <Avatar name={r.name} colorKey={r.color_key} size="sm" />
              <View className="flex-1">
                <Text variant="bodyStrong">{r.name}</Text>
                <Text variant="secondary">{rulesSummary(r.rules)}</Text>
              </View>
              {horse.can_manage ? <Icon as={ChevronRight} size={20} className="text-muted" /> : null}
            </>
          );
          return horse.can_manage ? (
            <PressableCard
              key={r.user_id}
              padded={false}
              onPress={() => edit(r)}
              accessibilityLabel={`${r.name} bearbeiten`}
              className="min-h-touch flex-row items-center gap-3 px-4 py-3"
            >
              {row}
            </PressableCard>
          ) : (
            <Card key={r.user_id} padded={false} className="min-h-touch flex-row items-center gap-3 px-4 py-3">
              {row}
            </Card>
          );
        })
      )}

      {horse.can_manage ? (
        <Button
          label="Reitbeteiligung hinzufügen"
          icon={UserPlus}
          variant="outline"
          onPress={() => {
            setError(null);
            setPicking(true);
          }}
        />
      ) : null}

      <Sheet
        open={picking}
        onOpenChange={setPicking}
        title="Reitbeteiligung hinzufügen"
        description="Wer aus dem Stall darf dieses Pferd reiten?"
      >
        {candidates.length === 0 ? (
          <Text variant="secondary">Alle anderen Personen im Stall sind schon eingetragen.</Text>
        ) : (
          <View>
            {candidates.map((m, i) => (
              <View key={m.id}>
                {i > 0 ? <Divider /> : null}
                <PressableCard
                  padded={false}
                  className="min-h-touch flex-row items-center gap-3 border-0 px-1 py-2"
                  accessibilityLabel={`${m.name} auswählen`}
                  onPress={() => {
                    setPicking(false);
                    setEditing({
                      userId: m.id,
                      name: m.name,
                      colorKey: m.color_key,
                      rules: [...DEFAULT_RIDER_RULES],
                      isNew: true,
                    });
                  }}
                >
                  <Avatar name={m.name} colorKey={m.color_key} size="sm" />
                  <Text variant="body">{m.name}</Text>
                </PressableCard>
              </View>
            ))}
          </View>
        )}
      </Sheet>

      <Sheet
        open={editing !== null}
        onOpenChange={(open) => !open && setEditing(null)}
        title={editing?.name ?? ""}
        description={`Was darf ${editing?.name ?? "die Person"} bei ${horse.name}?`}
      >
        {editing ? (
          <>
            <View>
              {RIDER_RULES.map((rule) => (
                <Switch
                  key={rule.key}
                  label={rule.label}
                  description={rule.description}
                  value={editing.rules.includes(rule.key)}
                  onValueChange={() => setEditing({ ...editing, rules: toggleRule(editing.rules, rule.key) })}
                />
              ))}
            </View>
            {error ? (
              <Text variant="bodySm" tone="danger" accessibilityRole="alert">
                {error}
              </Text>
            ) : null}
            <Button
              label={editing.isNew ? "Hinzufügen" : "Speichern"}
              fullWidth
              loading={save.isPending}
              onPress={() => run(() => save.mutateAsync({ userId: editing.userId, rules: editing.rules }))}
            />
            {!editing.isNew ? (
              <Button
                label="Reitbeteiligung entfernen"
                variant="ghost"
                fullWidth
                loading={remove.isPending}
                onPress={() => run(() => remove.mutateAsync(editing.userId))}
              />
            ) : null}
          </>
        ) : null}
      </Sheet>
    </View>
  );
}
